package pgstore

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/tern/migrate"
)

func wipeDB(ctx context.Context, t *testing.T, conn *pgx.Conn) {
	// Drop entire iidy schema.
	_, err := conn.Exec(ctx, fmt.Sprintf("drop schema if exists iidy cascade"))
	if err != nil {
		t.Fatalf("Could not destroy iidy schema : %v", err)
	}

	// Drop special tern table.
	_, err = conn.Exec(ctx, fmt.Sprintf("drop table if exists %s", TernDefaultMigrationTable))
	if err != nil {
		t.Fatalf("Could not drop tern migration table \"%s\": %v", TernDefaultMigrationTable, err)
	}
}

func migrateToLatest(ctx context.Context, t *testing.T, conn *pgx.Conn) {
	const ternDefaultMigrationTable string = "public.schema_version"
	migrator, err := migrate.NewMigrator(ctx, conn, ternDefaultMigrationTable)
	if err != nil {
		t.Fatalf("Could not create tern migrator: %v", err)
	}
	err = migrator.LoadMigrations("../migrations")
	if err != nil {
		t.Fatalf("Could not load tern migrations: %v", err)
	}
	err = migrator.Migrate(ctx)
	if err != nil {
		t.Fatalf("Could not run tern migration: %v", err)
	}
}

const DefaultTestMigrationConnectionURL string = "postgresql://postgres:postgres@localhost:5432/postgres?application_name=iidy_test_migration"
const DefaultTestPoolURL string = "postgresql://postgres:postgres@localhost:5432/postgres?pool_max_conns=1&application_name=iidy_test_suite"

func Test_PgStore(t *testing.T) {
	ctx := context.Background()

	conn, err := pgx.Connect(ctx, DefaultTestMigrationConnectionURL)
	if err != nil {
		t.Fatalf("Could not create pgx conn: %v", err)
	}
	defer conn.Close(ctx)

	// Ensure the db is in a known state.
	wipeDB(ctx, t, conn)
	// Clean up db when done.
	defer wipeDB(ctx, t, conn)

	// Put db in known state by migrating to latest.
	migrateToLatest(ctx, t, conn)

	s, err := NewPgStore(DefaultTestPoolURL)
	if err != nil {
		t.Errorf("Error instantiating PgStore: %v", err)
	}

	// Run these tests serially so that we always know
	// the state of the db.

	t.Run("InsertOne", func(t *testing.T) {
		count, err := s.InsertOne(context.Background(), "downloads", "kernel.tar.gz")
		if err != nil {
			t.Errorf("Error adding item: %v", err)
		}
		if count != 1 {
			t.Error("Did not properly add item to list.")
		}
	})

	t.Run("GetOne", func(t *testing.T) {
		attempts, ok, err := s.GetOne(context.Background(), "downloads", "kernel.tar.gz")
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
		_, ok, err := s.GetOne(context.Background(), "downloads", "I do not exist")
		if err != nil {
			t.Errorf("Error getting item: %v", err)
		}
		if ok {
			t.Error("List claims to return value that was not added to list.")
		}
	})

	t.Run("GetOne list does not exist", func(t *testing.T) {
		_, ok, err := s.GetOne(context.Background(), "I do not exist", "kernel.tar.gz")
		if err != nil {
			t.Errorf("Error getting item: %v", err)
		}
		if ok {
			t.Error("Non-existent list claims to return value.")
		}
	})

	t.Run("DeleteOne", func(t *testing.T) {
		count, err := s.DeleteOne(context.Background(), "downloads", "kernel.tar.gz")
		if err != nil {
			t.Errorf("Error trying to delete item from list: %v", err)
		}
		if count != 1 {
			t.Error("Did not properly delete item from list.")
		}
	})

	t.Run("GetOne should fail on deleted item", func(t *testing.T) {
		_, ok, err := s.GetOne(context.Background(), "downloads", "kernel.tar.gz")
		if err != nil {
			t.Errorf("Error getting item: %v", err)
		}
		if ok {
			t.Error("Did not properly delete item to list.")
		}
	})

	t.Run("DeleteOne item was not there in the first place", func(t *testing.T) {
		count, err := s.DeleteOne(context.Background(), "downloads", "I do not exist")
		if err != nil {
			t.Errorf("Error trying to delete item from list: %v", err)
		}
		if count != 0 {
			t.Error("Did not properly report non-deletion of item.")
		}
	})

	t.Run("DeleteOne list was not there in the first place", func(t *testing.T) {
		count, err := s.DeleteOne(context.Background(), "I do not exist", "kernel.tar.gz")
		if err != nil {
			t.Errorf("Error trying to delete item from non-existent list: %v", err)
		}
		if count != 0 {
			t.Error("Did not properly report non-deletion of item from no-existent list.")
		}
	})

	t.Run("InsertOne for incrementing", func(t *testing.T) {
		count, err := s.InsertOne(context.Background(), "downloads", "kernel.tar.gz")
		if err != nil {
			t.Errorf("Error adding item: %v", err)
		}
		if count != 1 {
			t.Error("Did not properly add item to list.")
		}
	})

	t.Run("IncrementOne", func(t *testing.T) {
		count, err := s.IncrementOne(context.Background(), "downloads", "kernel.tar.gz")
		if err != nil {
			t.Errorf("Error trying to increment: %v", err)
		}
		if count != 1 {
			t.Error("Did not properly increment.")
		}
	})

	t.Run("GetOne that has been incremented", func(t *testing.T) {
		attempts, ok, err := s.GetOne(context.Background(), "downloads", "kernel.tar.gz")
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
		count, err := s.IncrementOne(context.Background(), "downloads", "I do not exist")
		if err != nil {
			t.Errorf("Error trying to increment item from list: %v", err)
		}
		if count != 0 {
			t.Error("Did not properly report non-increment of item.")
		}
	})

	t.Run("IncrementOne list does not exist", func(t *testing.T) {
		count, err := s.IncrementOne(context.Background(), "I do not exist", "kernel.tar.gz")
		if err != nil {
			t.Errorf("Error trying to increment item from list: %v", err)
		}
		if count != 0 {
			t.Error("Did not properly report non-increment of item from non-existent list.")
		}
	})

	t.Run("DeleteOne Starting Fresh", func(t *testing.T) {
		count, err := s.DeleteOne(context.Background(), "downloads", "kernel.tar.gz")
		if err != nil {
			t.Errorf("Error trying to delete item from list: %v", err)
		}
		if count != 1 {
			t.Error("Did not properly delete item from list.")
		}
	})

	testFiles := []string{"kernel.tar.gz", "vim.tar.gz", "robots.txt"}

	t.Run("InsertBatch", func(t *testing.T) {
		count, err := s.InsertBatch(context.Background(), "downloads", testFiles)
		if err != nil {
			t.Errorf("Error batch inserting: %v", err)
		}
		if count != 3 {
			t.Errorf("Batch incremented wrong number of items. Expected 5, got %v", count)
		}

		// If we get the list items, do they exist?
		for _, file := range testFiles {
			attempts, ok, err := s.GetOne(context.Background(), "downloads", file)
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
		count, err := s.InsertBatch(context.Background(), "downloads", []string{})
		if err != nil {
			t.Errorf("Error batch inserting: %v", err)
		}
		if count != 0 {
			t.Errorf("Batch added wrong number of items. Expected 0, got %v", count)
		}
	})

	t.Run("DeleteBatch", func(t *testing.T) {
		count, err := s.DeleteBatch(context.Background(), "downloads", testFiles)
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
		count, err := s.InsertBatch(context.Background(), "downloads", files)
		if err != nil {
			t.Errorf("Error batch inserting: %v", err)
		}
		if count != 7 {
			t.Errorf("Batch added wrong number of items. Expected 5, got %v", count)
		}

		// Does batch delete work?
		count, err = s.DeleteBatch(context.Background(), "downloads", []string{"a", "b", "c", "d", "e"})
		if err != nil {
			t.Errorf("Error batch deleting: %v", err)
		}
		if count != 5 {
			t.Errorf("Batch deleted wrong number of items. Expected 5, got %v", count)
		}

		// If we look for the deleted items, are they correctly missing?
		for _, file := range []string{"a", "b", "c", "d", "e"} {
			_, ok, err := s.GetOne(context.Background(), "downloads", file)
			if err != nil {
				t.Errorf("Error getting item: %v", err)
			}
			if ok {
				t.Errorf("Found item %v that should have been deleted from list.", file)
			}
		}

		// Were other items left alone?
		for _, file := range []string{"f", "g"} {
			attempts, ok, err := s.GetOne(context.Background(), "downloads", file)
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
		count, err = s.DeleteBatch(context.Background(), "downloads", []string{"f", "g"})
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
		count, err := s.InsertBatch(context.Background(), "downloads", files)
		if err != nil {
			t.Errorf("Error batch inserting: %v", err)
		}
		if count != 7 {
			t.Errorf("Batch added wrong number of items. Expected 5, got %v", count)
		}

		var tests = []struct {
			afterItem string
			want      []ListEntry
		}{
			{"", []ListEntry{{"a", 0}, {"b", 0}}},
			{"b", []ListEntry{{"c", 0}, {"d", 0}}},
			{"d", []ListEntry{{"e", 0}, {"f", 0}}},
			{"f", []ListEntry{{"g", 0}}},
		}

		// If we batch get 2 items at a time, does everything work?
		for _, test := range tests {
			items, err := s.GetBatch(context.Background(), "downloads", test.afterItem, 2)
			if err != nil {
				t.Errorf("Error batch fetching: %v", err)
			}
			if !reflect.DeepEqual(test.want, items) {
				t.Errorf("Expected %v; got %v", test.want, items)
			}
		}

		// What if we batch get nothing?
		items, err := s.GetBatch(context.Background(), "downloads", "", 0)
		if err != nil {
			t.Errorf("Error batch deleting: %v", err)
		}
		if len(items) != 0 {
			t.Errorf("Batch get of nothing yeilded results!")
		}

		// Now just delete remaining, to clear for next test
		count, err = s.DeleteBatch(context.Background(), "downloads", files)
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
		count, err := s.InsertBatch(context.Background(), "downloads", files)
		if err != nil {
			t.Errorf("Error batch inserting: %v", err)
		}
		if count != 7 {
			t.Errorf("Batch added wrong number of items. Expected 5, got %v", count)
		}

		// Does batch increment work?
		count, err = s.IncrementBatch(context.Background(), "downloads", []string{"a", "b", "c", "d", "e"})
		if err != nil {
			t.Errorf("Error batch incrementing: %v", err)
		}
		if count != 5 {
			t.Errorf("Batch incremented wrong number of items. Expected 5, got %v", count)
		}

		// If we look for incremented items, are they incremented?
		for _, file := range []string{"a", "b", "c", "d", "e"} {
			attempts, ok, err := s.GetOne(context.Background(), "downloads", file)
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
			attempts, ok, err := s.GetOne(context.Background(), "downloads", file)
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
		count, err = s.IncrementBatch(context.Background(), "downloads", []string{})
		if err != nil {
			t.Errorf("Error batch deleting: %v", err)
		}
		if count != 0 {
			t.Errorf("Batch incremented wrong number of items. Expected 0, got %v", count)
		}

		// Now just delete remaining, to clear for next test
		count, err = s.DeleteBatch(context.Background(), "downloads", files)
		if err != nil {
			t.Errorf("Error batch deleting: %v", err)
		}
		if count != int64(len(files)) {
			t.Errorf("Batch deleted wrong number of items. Expected %d, got %v", len(files), count)
		}
	})

}
