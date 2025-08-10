package iidy

// These tests do ***NOT RUN BY DEFAULT*** when you type `go test ./...`
//
// WARNING: Running these integration tests ***WILL DESTROY YOUR DATABASE***.
//
// To run the integration test for data, you need 1) a locally-running
// PostgreSQL cluster, and 2) to set the following two env vars:
//
//	export INTEGTEST_DESTROY_DB_URL=postgres://postgres:postgres@localhost:5432/postgres
//	export INTEGTEST_DESTROY_DB_I_MEAN_IT=true
//
// To run the integration test for the server binary, you need to have 1) a locally-running
// PostgreSQL cluster, 2) the server running and pointed at port 9090, and 3) the following
// env vars set:
//
//	export INTEGTEST_DESTROY_DB_URL=postgres://postgres:postgres@localhost:5432/postgres
//	export INTEGTEST_DESTROY_DB_I_MEAN_IT=true
//	export INTEGTEST_SERVER=true
//
// NOTE that when Go runs tests, it runs every package in parallel (or, at least, it can).
// These integration tests follow the tip given here
// https://pkg.go.dev/testing@master#hdr-Subtests_and_Sub_benchmarks
// so that this entire "package" 1) is the ONLY source of integration tests (so other
// package tests can run in parallel because they have no state and cannot interfere
// with each other), and 2) runs all of its tests SERIALLY so that we can reason
// about the state of the database, which gets mutated during the test run.
// IF ANY OTHER INTEGRATION TESTS NEED TO MUTATE THE DATABASE, PLEASE ADD THOSE
// TESTS TO THIS FILE SO THAT INTEGRATION TESTS WILL CONTINUE TO RUN SERIALLY
// AND NOT TRIP OVER EACH OTHER!

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/manniwood/iidy/data"
	"github.com/manniwood/iidy/migrations"
)

// This is the only PUBLIC function that a run of
// `go test ./...` will find. All sub tests
// are run serially, in a controlled manner. However,
// use of t.Run() and other Go test facilities still
// makes this play nicely with Go's way of testing.
// See https://pkg.go.dev/testing@master#hdr-Subtests_and_Sub_benchmarks
// for more details.
func Test_Integrations(t *testing.T) {
	testDataFunctions(t)
	testServer(t)
}

func testDataFunctions(t *testing.T) {
	// INTEGTEST_DESTROY_DB_URL=postgres://postgres:postgres@localhost:5432/postgres
	dbURL := os.Getenv("INTEGTEST_DESTROY_DB_URL")
	if dbURL == "" {
		t.Skip("Not running data integration tests; INTEGTEST_DESTROY_DB_URL not set.")
	}

	ctx := context.Background()

	migrateConn, err := data.CreatePGXConnForMigration(ctx, dbURL)
	if err != nil {
		t.Fatalf("Could not create pgx conn for migration: %v", err)
	}
	defer migrateConn.Close(ctx)

	// Ensure the db is in a known state.
	wipeDB(ctx, t, migrateConn)
	// Clean up db when done.
	defer wipeDB(ctx, t, migrateConn)

	err = data.MigrateDB(ctx, migrateConn, migrations.Migrations, data.TernMigrationTable)
	if err != nil {
		t.Fatalf("Could not migrate db: %v", err)
	}

	pool, err := data.CreatePGXPool(ctx, dbURL)
	if err != nil {
		t.Fatalf("Could not create pgx pool for integ tests: %v", err)
	}
	defer pool.Close()

	// Run these tests serially so that we always know
	// the state of the db.

	t.Run("InsertOne", func(t *testing.T) {
		count, err := data.InsertOne(context.Background(), pool, "downloads", "kernel.tar.gz")
		if err != nil {
			t.Errorf("Error adding item: %v", err)
		}
		if count != 1 {
			t.Error("Did not properly add item to list.")
		}
	})

	t.Run("GetOne", func(t *testing.T) {
		attempts, ok, err := data.GetOne(context.Background(), pool, "downloads", "kernel.tar.gz")
		if err != nil {
			t.Errorf("Error getting item: %v", err)
		}
		if attempts != 0 {
			t.Error("attempts != 0")
		}
		if !ok {
			t.Error("Did not properly add item to list.")
		}
	})

	t.Run("GetOne item does not exist", func(t *testing.T) {
		_, ok, err := data.GetOne(context.Background(), pool, "downloads", "I do not exist")
		if err != nil {
			t.Errorf("Error getting item: %v", err)
		}
		if ok {
			t.Error("List claims to return value that was not added to list.")
		}
	})

	t.Run("GetOne list does not exist", func(t *testing.T) {
		_, ok, err := data.GetOne(context.Background(), pool, "I do not exist", "kernel.tar.gz")
		if err != nil {
			t.Errorf("Error getting item: %v", err)
		}
		if ok {
			t.Error("Non-existent list claims to return value.")
		}
	})

	t.Run("DeleteOne", func(t *testing.T) {
		count, err := data.DeleteOne(context.Background(), pool, "downloads", "kernel.tar.gz")
		if err != nil {
			t.Errorf("Error trying to delete item from list: %v", err)
		}
		if count != 1 {
			t.Error("Did not properly delete item from list.")
		}
	})

	t.Run("GetOne should fail on deleted item", func(t *testing.T) {
		_, ok, err := data.GetOne(context.Background(), pool, "downloads", "kernel.tar.gz")
		if err != nil {
			t.Errorf("Error getting item: %v", err)
		}
		if ok {
			t.Error("Did not properly delete item to list.")
		}
	})

	t.Run("DeleteOne item was not there in the first place", func(t *testing.T) {
		count, err := data.DeleteOne(context.Background(), pool, "downloads", "I do not exist")
		if err != nil {
			t.Errorf("Error trying to delete item from list: %v", err)
		}
		if count != 0 {
			t.Error("Did not properly report non-deletion of item.")
		}
	})

	t.Run("DeleteOne list was not there in the first place", func(t *testing.T) {
		count, err := data.DeleteOne(context.Background(), pool, "I do not exist", "kernel.tar.gz")
		if err != nil {
			t.Errorf("Error trying to delete item from non-existent list: %v", err)
		}
		if count != 0 {
			t.Error("Did not properly report non-deletion of item from no-existent list.")
		}
	})

	t.Run("InsertOne for incrementing", func(t *testing.T) {
		count, err := data.InsertOne(context.Background(), pool, "downloads", "kernel.tar.gz")
		if err != nil {
			t.Errorf("Error adding item: %v", err)
		}
		if count != 1 {
			t.Error("Did not properly add item to list.")
		}
	})

	t.Run("IncrementOne", func(t *testing.T) {
		count, err := data.IncrementOne(context.Background(), pool, "downloads", "kernel.tar.gz")
		if err != nil {
			t.Errorf("Error trying to increment: %v", err)
		}
		if count != 1 {
			t.Error("Did not properly increment.")
		}
	})

	t.Run("GetOne that has been incremented", func(t *testing.T) {
		attempts, ok, err := data.GetOne(context.Background(), pool, "downloads", "kernel.tar.gz")
		if err != nil {
			t.Errorf("Error getting item: %v", err)
		}
		if !ok {
			t.Error("Did not properly add item to list.")
		}
		if attempts != 1 {
			t.Error("Did not properly increment item in list.")
		}
	})

	t.Run("IncrementOne item does not exist", func(t *testing.T) {
		count, err := data.IncrementOne(context.Background(), pool, "downloads", "I do not exist")
		if err != nil {
			t.Errorf("Error trying to increment item from list: %v", err)
		}
		if count != 0 {
			t.Error("Did not properly report non-increment of item.")
		}
	})

	t.Run("IncrementOne list does not exist", func(t *testing.T) {
		count, err := data.IncrementOne(context.Background(), pool, "I do not exist", "kernel.tar.gz")
		if err != nil {
			t.Errorf("Error trying to increment item from list: %v", err)
		}
		if count != 0 {
			t.Error("Did not properly report non-increment of item from non-existent list.")
		}
	})

	t.Run("DeleteOne Starting Fresh", func(t *testing.T) {
		count, err := data.DeleteOne(context.Background(), pool, "downloads", "kernel.tar.gz")
		if err != nil {
			t.Errorf("Error trying to delete item from list: %v", err)
		}
		if count != 1 {
			t.Error("Did not properly delete item from list.")
		}
	})

	testFiles := []string{"kernel.tar.gz", "vim.tar.gz", "robots.txt"}

	t.Run("InsertBatch", func(t *testing.T) {
		count, err := data.InsertBatch(context.Background(), pool, "downloads", testFiles)
		if err != nil {
			t.Errorf("Error batch inserting: %v", err)
		}
		if count != 3 {
			t.Errorf("Batch incremented wrong number of items. Expected 5, got %v", count)
		}

		// If we get the list items, do they exist?
		for _, file := range testFiles {
			attempts, ok, err := data.GetOne(context.Background(), pool, "downloads", file)
			if err != nil {
				t.Errorf("Error getting item: %v", err)
			}
			if attempts != 0 {
				t.Errorf("Attempts for freshly-created %v is not 0", file)
			}
			if !ok {
				t.Error("Did not properly add item to list.")
			}
		}
	})

	t.Run("InsertBatch nothing", func(t *testing.T) {
		count, err := data.InsertBatch(context.Background(), pool, "downloads", []string{})
		if err != nil {
			t.Errorf("Error batch inserting: %v", err)
		}
		if count != 0 {
			t.Errorf("Batch added wrong number of items. Expected 0, got %v", count)
		}
	})

	t.Run("DeleteBatch", func(t *testing.T) {
		count, err := data.DeleteBatch(context.Background(), pool, "downloads", testFiles)
		if err != nil {
			t.Errorf("Error batch deleting: %v", err)
		}
		if count != int64(len(testFiles)) {
			t.Errorf("Batch deleted wrong number of items. Expected %d, got %v", len(testFiles), count)
		}
	})

	t.Run("DeleteBatch partial", func(t *testing.T) {
		// Batch add a bunch of test items.
		files := []string{"a", "b", "c", "d", "e", "f", "g"}
		count, err := data.InsertBatch(context.Background(), pool, "downloads", files)
		if err != nil {
			t.Errorf("Error batch inserting: %v", err)
		}
		if count != 7 {
			t.Errorf("Batch added wrong number of items. Expected 7, got %v", count)
		}

		// Does batch delete work?
		count, err = data.DeleteBatch(context.Background(), pool, "downloads", []string{"a", "b", "c", "d", "e"})
		if err != nil {
			t.Errorf("Error batch deleting: %v", err)
		}
		if count != 5 {
			t.Errorf("Batch deleted wrong number of items. Expected 5, got %v", count)
		}

		// If we look for the deleted items, are they correctly missing?
		for _, file := range []string{"a", "b", "c", "d", "e"} {
			_, ok, err := data.GetOne(context.Background(), pool, "downloads", file)
			if err != nil {
				t.Errorf("Error getting item: %v", err)
			}
			if ok {
				t.Errorf("Found item %v that should have been deleted from list.", file)
			}
		}

		// Were other items left alone?
		for _, file := range []string{"f", "g"} {
			attempts, ok, err := data.GetOne(context.Background(), pool, "downloads", file)
			if err != nil {
				t.Errorf("Error getting item: %v", err)
			}
			if !ok {
				t.Errorf("Item %v should not have been deleted from list.", file)
			}
			if attempts != 0 {
				t.Errorf("Item %v is incorrectly incremented.", file)
			}
		}

		// Now just delete remaining, to clear for next test
		count, err = data.DeleteBatch(context.Background(), pool, "downloads", []string{"f", "g"})
		if err != nil {
			t.Errorf("Error batch deleting: %v", err)
		}
		if count != 2 {
			t.Errorf("Batch deleted wrong number of items. Expected 2, got %v", count)
		}
	})

	t.Run("GetBatch", func(t *testing.T) {
		// Batch add a bunch of test items.
		files := []string{"a", "b", "c", "d", "e", "f", "g"}
		count, err := data.InsertBatch(context.Background(), pool, "downloads", files)
		if err != nil {
			t.Errorf("Error batch inserting: %v", err)
		}
		if count != 7 {
			t.Errorf("Batch added wrong number of items. Expected 5, got %v", count)
		}

		var tests = []struct {
			afterItem string
			want      []data.ListEntry
		}{
			{"", []data.ListEntry{{Item: "a", Attempts: 0}, {Item: "b", Attempts: 0}}},
			{"b", []data.ListEntry{{Item: "c", Attempts: 0}, {Item: "d", Attempts: 0}}},
			{"d", []data.ListEntry{{Item: "e", Attempts: 0}, {Item: "f", Attempts: 0}}},
			{"f", []data.ListEntry{{Item: "g", Attempts: 0}}},
		}

		// If we batch get 2 items at a time, does everything work?
		for _, test := range tests {
			items, err := data.GetBatch(context.Background(), pool, "downloads", test.afterItem, 2)
			if err != nil {
				t.Errorf("Error batch fetching: %v", err)
			}
			if !reflect.DeepEqual(test.want, items) {
				t.Errorf("Expected %v; got %v", test.want, items)
			}
		}

		// What if we batch get nothing?
		items, err := data.GetBatch(context.Background(), pool, "downloads", "", 0)
		if err != nil {
			t.Errorf("Error batch deleting: %v", err)
		}
		if len(items) != 0 {
			t.Errorf("Batch get of nothing yeilded results!")
		}

		// Now just delete remaining, to clear for next test
		count, err = data.DeleteBatch(context.Background(), pool, "downloads", files)
		if err != nil {
			t.Errorf("Error batch deleting: %v", err)
		}
		if count != int64(len(files)) {
			t.Errorf("Batch deleted wrong number of items. Expected %d, got %v", len(files), count)
		}
	})

	t.Run("IncrementBatch", func(t *testing.T) {
		// Batch add a bunch of test items.
		files := []string{"a", "b", "c", "d", "e", "f", "g"}
		count, err := data.InsertBatch(context.Background(), pool, "downloads", files)
		if err != nil {
			t.Errorf("Error batch inserting: %v", err)
		}
		if count != 7 {
			t.Errorf("Batch added wrong number of items. Expected 5, got %v", count)
		}

		// Does batch increment work?
		count, err = data.IncrementBatch(context.Background(), pool, "downloads", []string{"a", "b", "c", "d", "e"})
		if err != nil {
			t.Errorf("Error batch incrementing: %v", err)
		}
		if count != 5 {
			t.Errorf("Batch incremented wrong number of items. Expected 5, got %v", count)
		}

		// If we look for incremented items, are they incremented?
		for _, file := range []string{"a", "b", "c", "d", "e"} {
			attempts, ok, err := data.GetOne(context.Background(), pool, "downloads", file)
			if err != nil {
				t.Errorf("Error getting item: %v", err)
			}
			if !ok {
				t.Errorf("Did not properly get item %v from list.", file)
			}
			if attempts != 1 {
				t.Errorf("Did not properly increment item %v.", file)
			}
		}

		// What about non-incremented items? Were they left alone?
		for _, file := range []string{"f", "g"} {
			attempts, ok, err := data.GetOne(context.Background(), pool, "downloads", file)
			if err != nil {
				t.Errorf("Error getting item: %v", err)
			}
			if !ok {
				t.Errorf("Did not properly get item %v from list.", file)
			}
			if attempts != 0 {
				t.Errorf("Item %v is incorrectly incremented.", file)
			}
		}

		// What if we batch increment nothing?
		count, err = data.IncrementBatch(context.Background(), pool, "downloads", []string{})
		if err != nil {
			t.Errorf("Error batch deleting: %v", err)
		}
		if count != 0 {
			t.Errorf("Batch incremented wrong number of items. Expected 0, got %v", count)
		}

		// Now just delete remaining, to clear for next test
		count, err = data.DeleteBatch(context.Background(), pool, "downloads", files)
		if err != nil {
			t.Errorf("Error batch deleting: %v", err)
		}
		if count != int64(len(files)) {
			t.Errorf("Batch deleted wrong number of items. Expected %d, got %v", len(files), count)
		}
	})
}

func testServer(t *testing.T) {
	runServerTests := os.Getenv("INTEGTEST_SERVER")
	if runServerTests != "true" {
		t.Skip("Not running server integration tests; INTEGTEST_SERVER != true.")
	}
	// This is a little strange. Even though the server migrates the db
	// at startup, we want to be sure the db is in a known state, so we
	// wipe it clean and do a fresh migration. This essentially "pulls the
	// rug out from under the service", but because we are the only client
	// of the service, we can do this.

	// INTEGTEST_DESTROY_DB_URL=postgres://postgres:postgres@localhost:5432/postgres
	dbURL := os.Getenv("INTEGTEST_DESTROY_DB_URL")
	if dbURL == "" {
		t.Skip("Not running data integration tests; INTEGTEST_DESTROY_DB_URL not set.")
	}

	ctx := context.Background()

	migrateConn, err := data.CreatePGXConnForMigration(ctx, dbURL)
	if err != nil {
		t.Fatalf("Could not create pgx conn for migration: %v", err)
	}
	defer migrateConn.Close(ctx)

	// Ensure the db is in a known state.
	wipeDB(ctx, t, migrateConn)
	// Clean up db when done.
	defer wipeDB(ctx, t, migrateConn)

	err = data.MigrateDB(ctx, migrateConn, migrations.Migrations, data.TernMigrationTable)
	if err != nil {
		t.Fatalf("Could not migrate db: %v", err)
	}

	// Run these tests serially so that we always know
	// the state of the db behind the service.

	t.Run("InsertOne", func(t *testing.T) {
		resp, err := http.Post(
			"http://localhost:8080/iidy/v1/lists/downloads/kernel.tar.gz",
			"text/plain",
			nil)
		if err != nil {
			t.Errorf("Error adding item: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			t.Errorf("Expected status code %d but got: %v", http.StatusCreated, resp.StatusCode)
		}

		body, err := readAllHttpResponseBody(resp.Body)
		if err != nil {
			t.Errorf("Error reading http response body: %v", err)
		}
		if body != "ADDED 1\n" {
			t.Errorf("http response body was supposed to be ADDED 1 but was %s instead", body)
		}
	})

	t.Run("GetOne", func(t *testing.T) {
		resp, err := http.Get("http://localhost:8080/iidy/v1/lists/downloads/kernel.tar.gz")
		if err != nil {
			t.Errorf("Error getting item: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status code %d but got: %v", http.StatusOK, resp.StatusCode)
		}

		body, err := readAllHttpResponseBody(resp.Body)
		if err != nil {
			t.Errorf("Error reading http response body: %v", err)
		}
		if body != "0\n" {
			t.Errorf("http response body was supposed to be 0 but was %s instead", body)
		}
	})

	t.Run("GetOne item does not exist", func(t *testing.T) {
		resp, err := http.Get("http://localhost:8080/iidy/v1/lists/downloads/I_do_not_exist")
		if err != nil {
			t.Errorf("Error getting item: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status code %d but got: %v", http.StatusNotFound, resp.StatusCode)
		}
	})

	t.Run("GetOne list does not exist", func(t *testing.T) {
		resp, err := http.Get("http://localhost:8080/iidy/v1/lists/I_do_not_exist/kernel.tar.gz")
		if err != nil {
			t.Errorf("Error getting item: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status code %d but got: %v", http.StatusNotFound, resp.StatusCode)
		}
	})

	t.Run("DeleteOne", func(t *testing.T) {
		req, err := http.NewRequest(
			"DELETE",
			"http://localhost:8080/iidy/v1/lists/downloads/kernel.tar.gz",
			nil)
		if err != nil {
			t.Errorf("Error trying to make new http request: %v", err)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Errorf("Error deleting item: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status code %d but got: %v", http.StatusOK, resp.StatusCode)
		}

		body, err := readAllHttpResponseBody(resp.Body)
		if err != nil {
			t.Errorf("Error reading http response body: %v", err)
		}
		if body != "DELETED 1\n" {
			t.Errorf("http response body was supposed to be DELETED 1 but was %s instead", body)
		}
	})

	t.Run("GetOne should fail on deleted item", func(t *testing.T) {
		resp, err := http.Get("http://localhost:8080/iidy/v1/lists/downloads/kernel.tar.gz")
		if err != nil {
			t.Errorf("Error getting item: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status code %d but got: %v", http.StatusNotFound, resp.StatusCode)
		}
	})

	t.Run("DeleteOne item was not there in the first place", func(t *testing.T) {
		req, err := http.NewRequest(
			"DELETE",
			"http://localhost:8080/iidy/v1/lists/downloads/I_do_not_exist",
			nil)
		if err != nil {
			t.Errorf("Error trying to make new http request: %v", err)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Errorf("Error deleting item: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status code %d but got: %v", http.StatusOK, resp.StatusCode)
		}

		body, err := readAllHttpResponseBody(resp.Body)
		if err != nil {
			t.Errorf("Error reading http response body: %v", err)
		}
		if body != "DELETED 0\n" {
			t.Errorf("http response body was supposed to be DELETED 0 but was %s instead", body)
		}
	})

	t.Run("DeleteOne list was not there in the first place", func(t *testing.T) {
		req, err := http.NewRequest(
			"DELETE",
			"http://localhost:8080/iidy/v1/lists/I_do_not_exist/kernel.tar.gz",
			nil)
		if err != nil {
			t.Errorf("Error trying to make new http request: %v", err)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Errorf("Error deleting item: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status code %d but got: %v", http.StatusOK, resp.StatusCode)
		}

		body, err := readAllHttpResponseBody(resp.Body)
		if err != nil {
			t.Errorf("Error reading http response body: %v", err)
		}
		if body != "DELETED 0\n" {
			t.Errorf("http response body was supposed to be DELETED 0 but was %s instead", body)
		}
	})

	t.Run("InsertOne for incrementing", func(t *testing.T) {
		resp, err := http.Post(
			"http://localhost:8080/iidy/v1/lists/downloads/kernel.tar.gz",
			"text/plain",
			nil)
		if err != nil {
			t.Errorf("Error adding item: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			t.Errorf("Expected status code %d but got: %v", http.StatusCreated, resp.StatusCode)
		}
	})

	t.Run("IncrementOne", func(t *testing.T) {
		resp, err := http.Post(
			"http://localhost:8080/iidy/v1/increment/lists/downloads/kernel.tar.gz",
			"text/plain",
			nil)
		if err != nil {
			t.Errorf("Error incrementing item: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status code %d but got: %v", http.StatusOK, resp.StatusCode)
		}

		body, err := readAllHttpResponseBody(resp.Body)
		if err != nil {
			t.Errorf("Error reading http response body: %v", err)
		}
		if body != "INCREMENTED 1\n" {
			t.Errorf("http response body was supposed to be INCREMENTED 1 but was %s instead", body)
		}
	})

	t.Run("GetOne that has been incremented", func(t *testing.T) {
		resp, err := http.Get("http://localhost:8080/iidy/v1/lists/downloads/kernel.tar.gz")
		if err != nil {
			t.Errorf("Error getting item: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status code %d but got: %v", http.StatusOK, resp.StatusCode)
		}

		body, err := readAllHttpResponseBody(resp.Body)
		if err != nil {
			t.Errorf("Error reading http response body: %v", err)
		}
		if body != "1\n" {
			t.Errorf("http response body was supposed to be 1 but was %s instead", body)
		}
	})

	t.Run("IncrementOne item that does not exist", func(t *testing.T) {
		resp, err := http.Post(
			"http://localhost:8080/iidy/v1/increment/lists/downloads/I_do_not_exist",
			"text/plain",
			nil)
		if err != nil {
			t.Errorf("Error incrementing item: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status code %d but got: %v", http.StatusOK, resp.StatusCode)
		}

		body, err := readAllHttpResponseBody(resp.Body)
		if err != nil {
			t.Errorf("Error reading http response body: %v", err)
		}
		if body != "INCREMENTED 0\n" {
			t.Errorf("http response body was supposed to be INCREMENTED 0 but was %s instead", body)
		}
	})

	t.Run("IncrementOne where list does not exist", func(t *testing.T) {
		resp, err := http.Post(
			"http://localhost:8080/iidy/v1/increment/lists/I_do_not_exist/kernel.tar.gz",
			"text/plain",
			nil)
		if err != nil {
			t.Errorf("Error incrementing item: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status code %d but got: %v", http.StatusOK, resp.StatusCode)
		}

		body, err := readAllHttpResponseBody(resp.Body)
		if err != nil {
			t.Errorf("Error reading http response body: %v", err)
		}
		if body != "INCREMENTED 0\n" {
			t.Errorf("http response body was supposed to be INCREMENTED 0 but was %s instead", body)
		}
	})

	t.Run("DeleteOne Starting Fresh", func(t *testing.T) {
		req, err := http.NewRequest(
			"DELETE",
			"http://localhost:8080/iidy/v1/lists/downloads/kernel.tar.gz",
			nil)
		if err != nil {
			t.Errorf("Error trying to make new http request: %v", err)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Errorf("Error deleting item: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status code %d but got: %v", http.StatusOK, resp.StatusCode)
		}

		body, err := readAllHttpResponseBody(resp.Body)
		if err != nil {
			t.Errorf("Error reading http response body: %v", err)
		}
		if body != "DELETED 1\n" {
			t.Errorf("http response body was supposed to be DELETED 1 but was %s instead", body)
		}
	})

	testFiles := []string{"kernel.tar.gz", "vim.tar.gz", "robots.txt"}

	t.Run("InsertBatch", func(t *testing.T) {
		testFilesBody := strings.Join(testFiles, "\n")
		resp, err := http.Post(
			"http://localhost:8080/iidy/v1/batch/lists/downloads",
			"text/plain",
			bytes.NewBufferString(testFilesBody))
		if err != nil {
			t.Errorf("Error adding items: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			t.Errorf("Expected status code %d but got: %v", http.StatusCreated, resp.StatusCode)
		}

		body, err := readAllHttpResponseBody(resp.Body)
		if err != nil {
			t.Errorf("Error reading http response body: %v", err)
		}
		if body != "ADDED 3\n" {
			t.Errorf("http response body was supposed to be ADDED 3 but was %s instead", body)
		}
	})

	t.Run("InsertBatch nothing", func(t *testing.T) {
		resp, err := http.Post(
			"http://localhost:8080/iidy/v1/batch/lists/downloads",
			"text/plain",
			nil)
		if err != nil {
			t.Errorf("Error adding items: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			t.Errorf("Expected status code %d but got: %v", http.StatusCreated, resp.StatusCode)
		}

		body, err := readAllHttpResponseBody(resp.Body)
		if err != nil {
			t.Errorf("Error reading http response body: %v", err)
		}
		if body != "ADDED 0\n" {
			t.Errorf("http response body was supposed to be ADDED 0 but was %s instead", body)
		}
	})

	t.Run("DeleteBatch", func(t *testing.T) {
		testFilesBody := strings.Join(testFiles, "\n")
		req, err := http.NewRequest(
			"DELETE",
			"http://localhost:8080/iidy/v1/batch/lists/downloads",
			bytes.NewBufferString(testFilesBody))
		if err != nil {
			t.Errorf("Error trying to make new http request: %v", err)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Errorf("Error deleting items: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status code %d but got: %v", http.StatusOK, resp.StatusCode)
		}

		body, err := readAllHttpResponseBody(resp.Body)
		if err != nil {
			t.Errorf("Error reading http response body: %v", err)
		}
		if body != "DELETED 3\n" {
			t.Errorf("http response body was supposed to be DELETED 3 but was %s instead", body)
		}
	})

	t.Run("DeleteBatch partial", func(t *testing.T) {
		// Batch add a bunch of test items.
		files := []string{"a", "b", "c", "d", "e", "f", "g"}
		testFilesBody := strings.Join(files, "\n")
		resp, err := http.Post(
			"http://localhost:8080/iidy/v1/batch/lists/downloads",
			"text/plain",
			bytes.NewBufferString(testFilesBody))
		if err != nil {
			t.Errorf("Error adding items: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			t.Errorf("Expected status code %d but got: %v", http.StatusCreated, resp.StatusCode)
		}

		body, err := readAllHttpResponseBody(resp.Body)
		if err != nil {
			t.Errorf("Error reading http response body: %v", err)
		}
		if body != "ADDED 7\n" {
			t.Errorf("http response body was supposed to be ADDED 7 but was %s instead", body)
		}

		// Does batch delete work?
		files2 := []string{"a", "b", "c", "d", "e"}
		testFilesBody2 := strings.Join(files2, "\n")
		req2, err := http.NewRequest(
			"DELETE",
			"http://localhost:8080/iidy/v1/batch/lists/downloads",
			bytes.NewBufferString(testFilesBody2))
		if err != nil {
			t.Errorf("Error trying to make new http request: %v", err)
		}
		resp2, err := http.DefaultClient.Do(req2)
		if err != nil {
			t.Errorf("Error deleting items: %v", err)
		}
		defer resp2.Body.Close()

		if resp2.StatusCode != http.StatusOK {
			t.Errorf("Expected status code %d but got: %v", http.StatusOK, resp2.StatusCode)
		}

		body2, err := readAllHttpResponseBody(resp2.Body)
		if err != nil {
			t.Errorf("Error reading http response body: %v", err)
		}
		if body2 != "DELETED 5\n" {
			t.Errorf("http response body was supposed to be DELETED 5 but was %s instead", body2)
		}

		// If we look for the deleted items, are they correctly missing?
		for _, file := range []string{"a", "b", "c", "d", "e"} {
			resp3, err := http.Get("http://localhost:8080/iidy/v1/lists/downloads/" + file)
			if err != nil {
				t.Errorf("Error getting item: %v", err)
			}
			defer resp3.Body.Close()

			if resp3.StatusCode != http.StatusNotFound {
				t.Errorf("Expected status code %d but got: %v", http.StatusNotFound, resp3.StatusCode)
			}

			body3, err := readAllHttpResponseBody(resp3.Body)
			if err != nil {
				t.Errorf("Error reading http response body: %v", err)
			}
			if body3 != "Not found.\n" {
				t.Errorf("http response body was supposed to be empty but was %s instead", body3)
			}
		}

		// Were other items left alone?
		for _, file := range []string{"f", "g"} {
			resp4, err := http.Get("http://localhost:8080/iidy/v1/lists/downloads/" + file)
			if err != nil {
				t.Errorf("Error getting item: %v", err)
			}
			defer resp4.Body.Close()

			if resp4.StatusCode != http.StatusOK {
				t.Errorf("Expected status code %d but got: %v", http.StatusOK, resp4.StatusCode)
			}

			body4, err := readAllHttpResponseBody(resp4.Body)
			if err != nil {
				t.Errorf("Error reading http response body: %v", err)
			}
			if body4 != "0\n" {
				t.Errorf("http response body was supposed to be 0 but was %s instead", body4)
			}
		}

		// Now just delete remaining, to clear for next test
		files5 := []string{"f", "g"}
		testFilesBody5 := strings.Join(files5, "\n")
		req5, err := http.NewRequest(
			"DELETE",
			"http://localhost:8080/iidy/v1/batch/lists/downloads",
			bytes.NewBufferString(testFilesBody5))
		if err != nil {
			t.Errorf("Error trying to make new http request: %v", err)
		}
		resp5, err := http.DefaultClient.Do(req5)
		if err != nil {
			t.Errorf("Error deleting items: %v", err)
		}
		defer resp5.Body.Close()

		if resp5.StatusCode != http.StatusOK {
			t.Errorf("Expected status code %d but got: %v", http.StatusOK, resp5.StatusCode)
		}

		body5, err := readAllHttpResponseBody(resp5.Body)
		if err != nil {
			t.Errorf("Error reading http response body: %v", err)
		}
		if body5 != "DELETED 2\n" {
			t.Errorf("http response body was supposed to be DELETED 2 but was %s instead", body5)
		}
	})

	t.Run("GetBatch", func(t *testing.T) {
		// Batch add a bunch of test items.
		files := []string{"a", "b", "c", "d", "e", "f", "g"}
		testFilesBody := strings.Join(files, "\n")
		resp, err := http.Post(
			"http://localhost:8080/iidy/v1/batch/lists/downloads",
			"text/plain",
			bytes.NewBufferString(testFilesBody))
		if err != nil {
			t.Errorf("Error adding items: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			t.Errorf("Expected status code %d but got: %v", http.StatusCreated, resp.StatusCode)
		}

		body, err := readAllHttpResponseBody(resp.Body)
		if err != nil {
			t.Errorf("Error reading http response body: %v", err)
		}
		if body != "ADDED 7\n" {
			t.Errorf("http response body was supposed to be ADDED 7 but was %s instead", body)
		}

		var tests = []struct {
			afterItem string
			want      string
		}{
			{"", "a 0\nb 0\n"},
			{"b", "c 0\nd 0\n"},
			{"d", "e 0\nf 0\n"},
			{"f", "g 0\n"},
		}

		// If we batch get 2 items at a time, does everything work?
		for _, test := range tests {
			resp, err := http.Get("http://localhost:8080/iidy/v1/batch/lists/downloads?count=2&after_id=" + test.afterItem)
			if err != nil {
				t.Errorf("Error getting item: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected status code %d but got: %v", http.StatusOK, resp.StatusCode)
			}

			body, err := readAllHttpResponseBody(resp.Body)
			if err != nil {
				t.Errorf("Error reading http response body: %v", err)
			}
			if body != test.want {
				t.Errorf("http response body was supposed to be %s but was %s instead", test.want, body)
			}
		}

		// What if we batch get nothing?
		resp2, err := http.Get("http://localhost:8080/iidy/v1/batch/lists/downloads?count=0")
		if err != nil {
			t.Errorf("Error getting item: %v", err)
		}
		defer resp2.Body.Close()

		if resp2.StatusCode != http.StatusOK {
			t.Errorf("Expected status code %d but got: %v", http.StatusOK, resp.StatusCode)
		}

		body2, err := readAllHttpResponseBody(resp2.Body)
		if err != nil {
			t.Errorf("Error reading http response body: %v", err)
		}
		if body2 != "" {
			t.Errorf("http response body was supposed to be empty but was %s instead", body)
		}

		// Now just delete remaining, to clear for next test
		files5 := []string{"a", "b", "c", "d", "e", "f", "g"}
		testFilesBody5 := strings.Join(files5, "\n")
		req5, err := http.NewRequest(
			"DELETE",
			"http://localhost:8080/iidy/v1/batch/lists/downloads",
			bytes.NewBufferString(testFilesBody5))
		if err != nil {
			t.Errorf("Error trying to make new http request: %v", err)
		}
		resp5, err := http.DefaultClient.Do(req5)
		if err != nil {
			t.Errorf("Error deleting items: %v", err)
		}
		defer resp5.Body.Close()

		if resp5.StatusCode != http.StatusOK {
			t.Errorf("Expected status code %d but got: %v", http.StatusOK, resp5.StatusCode)
		}

		body5, err := readAllHttpResponseBody(resp5.Body)
		if err != nil {
			t.Errorf("Error reading http response body: %v", err)
		}
		if body5 != "DELETED 7\n" {
			t.Errorf("http response body was supposed to be DELETED 7 but was %s instead", body5)
		}
	})

	t.Run("IncrementBatch", func(t *testing.T) {
		// Batch add a bunch of test items.
		files := []string{"a", "b", "c", "d", "e", "f", "g"}
		testFilesBody := strings.Join(files, "\n")
		resp, err := http.Post(
			"http://localhost:8080/iidy/v1/batch/lists/downloads",
			"text/plain",
			bytes.NewBufferString(testFilesBody))
		if err != nil {
			t.Errorf("Error adding items: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			t.Errorf("Expected status code %d but got: %v", http.StatusCreated, resp.StatusCode)
		}

		body, err := readAllHttpResponseBody(resp.Body)
		if err != nil {
			t.Errorf("Error reading http response body: %v", err)
		}
		if body != "ADDED 7\n" {
			t.Errorf("http response body was supposed to be ADDED 7 but was %s instead", body)
		}

		// Does batch increment work?
		files2 := []string{"a", "b", "c", "d", "e"}
		testFilesBody2 := strings.Join(files2, "\n")
		resp2, err := http.Post(
			"http://localhost:8080/iidy/v1/increment/batch/lists/downloads",
			"text/plain",
			bytes.NewBufferString(testFilesBody2))
		if err != nil {
			t.Errorf("Error incrementing items: %v", err)
		}
		defer resp2.Body.Close()

		if resp2.StatusCode != http.StatusOK {
			t.Errorf("Expected status code %d but got: %v", http.StatusOK, resp2.StatusCode)
		}

		body2, err := readAllHttpResponseBody(resp2.Body)
		if err != nil {
			t.Errorf("Error reading http response body: %v", err)
		}
		if body2 != "INCREMENTED 5\n" {
			t.Errorf("http response body was supposed to be INCREMENTED 5 but was %s instead", body2)
		}

		// If we look for incremented items, are they incremented?
		for _, file := range []string{"a", "b", "c", "d", "e"} {
			resp, err := http.Get("http://localhost:8080/iidy/v1/lists/downloads/" + file)
			if err != nil {
				t.Errorf("Error getting item: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected status code %d but got: %v", http.StatusOK, resp.StatusCode)
			}

			body, err := readAllHttpResponseBody(resp.Body)
			if err != nil {
				t.Errorf("Error reading http response body: %v", err)
			}
			if body != "1\n" {
				t.Errorf("http response body was supposed to be 1 but was %s instead", body)
			}
		}

		// What about non-incremented items? Were they left alone?
		for _, file := range []string{"f", "g"} {
			resp, err := http.Get("http://localhost:8080/iidy/v1/lists/downloads/" + file)
			if err != nil {
				t.Errorf("Error getting item: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected status code %d but got: %v", http.StatusOK, resp.StatusCode)
			}

			body, err := readAllHttpResponseBody(resp.Body)
			if err != nil {
				t.Errorf("Error reading http response body: %v", err)
			}
			if body != "0\n" {
				t.Errorf("http response body was supposed to be 0 but was %s instead", body)
			}
		}

		// What if we batch increment nothing?
		resp3, err := http.Post(
			"http://localhost:8080/iidy/v1/increment/batch/lists/downloads",
			"text/plain",
			bytes.NewBufferString(""))
		if err != nil {
			t.Errorf("Error incrementing items: %v", err)
		}
		defer resp3.Body.Close()

		if resp3.StatusCode != http.StatusOK {
			t.Errorf("Expected status code %d but got: %v", http.StatusOK, resp3.StatusCode)
		}

		body3, err := readAllHttpResponseBody(resp3.Body)
		if err != nil {
			t.Errorf("Error reading http response body: %v", err)
		}
		if body3 != "INCREMENTED 0\n" {
			t.Errorf("http response body was supposed to be INCREMENTED 0 but was %s instead", body3)
		}

		// Now just delete remaining, to clear for next test
		files5 := []string{"a", "b", "c", "d", "e", "f", "g"}
		testFilesBody5 := strings.Join(files5, "\n")
		req5, err := http.NewRequest(
			"DELETE",
			"http://localhost:8080/iidy/v1/batch/lists/downloads",
			bytes.NewBufferString(testFilesBody5))
		if err != nil {
			t.Errorf("Error trying to make new http request: %v", err)
		}
		resp5, err := http.DefaultClient.Do(req5)
		if err != nil {
			t.Errorf("Error deleting items: %v", err)
		}
		defer resp5.Body.Close()

		if resp5.StatusCode != http.StatusOK {
			t.Errorf("Expected status code %d but got: %v", http.StatusOK, resp5.StatusCode)
		}

		body5, err := readAllHttpResponseBody(resp5.Body)
		if err != nil {
			t.Errorf("Error reading http response body: %v", err)
		}
		if body5 != "DELETED 7\n" {
			t.Errorf("http response body was supposed to be DELETED 7 but was %s instead", body5)
		}
	})
}

func readAllHttpResponseBody(body io.ReadCloser) (string, error) {
	bodyBytes, err := io.ReadAll(body)
	if err != nil {
		return "", err
	}
	return string(bodyBytes), nil
}

func wipeDB(ctx context.Context, t *testing.T, conn *pgx.Conn) {
	nukeDBOK := os.Getenv("INTEGTEST_DESTROY_DB_I_MEAN_IT")
	if nukeDBOK != "true" {
		t.Skip("Not running integration tests and not destroying db; INTEGTEST_DESTROY_DB_I_MEAN_IT != true.")
	}

	// Drop entire iidy schema.
	_, err := conn.Exec(ctx, fmt.Sprintf("drop schema if exists iidy cascade"))
	if err != nil {
		t.Fatalf("Could not destroy iidy schema : %v", err)
	}

	// Drop special tern table.
	_, err = conn.Exec(ctx, fmt.Sprintf("drop table if exists %s", data.TernMigrationTable))
	if err != nil {
		t.Fatalf("Could not drop tern migration table \"%s\": %v", data.TernMigrationTable, err)
	}
}
