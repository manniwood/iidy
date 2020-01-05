package iidy

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestPutHandler(t *testing.T) {
	req, err := http.NewRequest("PUT", "/lists/downloads/kernel.tar.gz", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	h := &Handler{Store: getEmptyStore(t)}
	handler := http.Handler(h)
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
	expected := "ADDED 1\n"
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
	}

	// Did we really add the item?
	_, ok, err := h.Store.Get(context.Background(), "downloads", "kernel.tar.gz")
	if err != nil {
		t.Errorf("Error getting item: %v", err)
	}
	if !ok {
		t.Error("Did not properly get item from list.")
	}
}

func TestNonExistentMethod(t *testing.T) {
	req, err := http.NewRequest("BLARG", "/lists/downloads/kernel.tar.gz", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	h := &Handler{Store: getEmptyStore(t)}
	handler := http.Handler(h)
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
	expected := "Unknown method.\n"
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
	}
}

func TestGetHandler(t *testing.T) {
	h := &Handler{Store: getEmptyStore(t)}
	addSingleStartingItem(t, h.Store)

	// Can we get an existing value?
	req, err := http.NewRequest("GET", "/lists/downloads/kernel.tar.gz", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	handler := http.Handler(h)
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
	expected := "0\n"
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
	}
}

func TestUnhappyHandlerGetScenarios(t *testing.T) {
	h := &Handler{Store: getEmptyStore(t)}
	addSingleStartingItem(t, h.Store)

	// What about getting an item that doesn't exist?
	req, err := http.NewRequest("GET", "/lists/downloads/i_do_not_exist.tar.gz", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	handler := http.Handler(h)
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
	expected := "Not found.\n"
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
	}

	// What about getting from a list that doesn't exist?
	req, err = http.NewRequest("GET", "/lists/i_do_not_exist/kernel.tar.gz", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
	}
}

func TestIncHandler(t *testing.T) {
	h := &Handler{Store: getEmptyStore(t)}
	addSingleStartingItem(t, h.Store)

	// Can we increment the number of attempts for a list item?
	req, err := http.NewRequest("INCREMENT", "/lists/downloads/kernel.tar.gz", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	handler := http.Handler(h)
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
	expected := "INCREMENTED 1\n"
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
	}

	// Is the incremented attempt fetchable with GET?
	req, err = http.NewRequest("GET", "/lists/downloads/kernel.tar.gz", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
	expected = "1\n"
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
	}

	// How about incrementing something that's not there?
	req, err = http.NewRequest("INCREMENT", "/lists/i_do_not_exist/kernel.tar.gz", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr = httptest.NewRecorder()
	handler = http.Handler(h)
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
	expected = "INCREMENTED 0\n"
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
	}

}

func TestDelHandler(t *testing.T) {
	h := &Handler{Store: getEmptyStore(t)}
	addSingleStartingItem(t, h.Store)

	// Can we delete our starting value?
	req, err := http.NewRequest("DELETE", "/lists/downloads/kernel.tar.gz", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	handler := http.Handler(h)
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
	expected := "DELETED 1\n"
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
	}

	// Is the deleted value really no longer fetchable?
	req, err = http.NewRequest("GET", "/lists/downloads/kernel.tar.gz", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
	expected = "Not found.\n"
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got '%v' want '%v'", rr.Body.String(), expected)
	}

	// How about deleting something that's not there?
	req, err = http.NewRequest("DELETE", "/lists/downloads/kernel.tar.gz", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr = httptest.NewRecorder()
	handler = http.Handler(h)
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
	expected = "DELETED 0\n"
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
	}

}

func TestBulkPutHandler(t *testing.T) {
	body := []byte(`kernel.tar.gz
vim.tar.gz
robots.txt`)
	// remember, these come back in alphabetical order
	expected := []ListEntry{
		{"kernel.tar.gz", 0},
		{"robots.txt", 0},
		{"vim.tar.gz", 0},
	}

	// Does bulk put work without errors?
	req, err := http.NewRequest("BULKPUT", "/lists/downloads", bytes.NewBuffer(body))
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	h := &Handler{Store: getEmptyStore(t)}
	handler := http.Handler(h)
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
	expectedBody := "ADDED 3\n"
	if rr.Body.String() != expectedBody {
		t.Errorf("Unexpected body: got %v want %v", rr.Body.String(), expectedBody)
	}

	// What if we bulk get what we just bulk put?
	listEntries, err := h.Store.BulkGet(context.Background(), "downloads", "", 3)
	if err != nil {
		t.Errorf("Error fetching items: %v", err)
	}
	if !reflect.DeepEqual(expected, listEntries) {
		t.Errorf("Expected %v; got %v", expected, listEntries)
	}

	// What if we bulk put nothing?
	req, err = http.NewRequest("BULKPUT", "/lists/downloads", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr = httptest.NewRecorder()
	h = &Handler{Store: getEmptyStore(t)} // XXX: needed?
	handler = http.Handler(h)             // XXX: needed?
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
	expectedBody = "ADDED 0\n"
	if rr.Body.String() != expectedBody {
		t.Errorf("Unexpected body: got %v want %v", rr.Body.String(), expectedBody)
	}
}

func TestBulkGetHandler(t *testing.T) {
	s := getEmptyStore(t)
	bulkAddTestItems(t, s)

	// Can we bulk get the test items in batches of 2?
	var tests = []struct {
		afterItem string
		want      string
		lastItem  string
	}{
		{"", "a 0\nb 0\n", "b"},
		{"b", "c 0\nd 0\n", "d"},
		{"d", "e 0\nf 0\n", "f"},
		{"f", "g 0\n", "g"},
	}
	for _, test := range tests {
		req, err := http.NewRequest("BULKGET", "/lists/downloads", nil)
		if err != nil {
			t.Fatal(err)
		}
		if test.afterItem != "" {
			req.Header.Set("X-IIDY-After-Item", test.afterItem)
		}
		req.Header.Set("X-IIDY-Count", "2")
		rr := httptest.NewRecorder()
		h := &Handler{Store: s}
		handler := http.Handler(h)
		handler.ServeHTTP(rr, req)
		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}
		lastItem := rr.Result().Header.Get("X-IIDY-Last-Item")
		if lastItem != test.lastItem {
			t.Errorf("handler returned wrong last item: got %v want %v", lastItem, test.lastItem)
		}
		if rr.Body.String() != test.want {
			t.Errorf("handler returned unexpected body: got '%v' want '%v'", rr.Body.String(), test.want)
		}
	}

	// What if we bulk get from a list that doesn't exist?
	req, err := http.NewRequest("BULKGET", "/lists/i_do_not_exist", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-IIDY-Count", "2")
	rr := httptest.NewRecorder()
	h := &Handler{Store: s}
	handler := http.Handler(h)
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
	/* lastItem := rr.Result().Header.Get("X-IIDY-Last-Item")

	if lastItem != test.lastItem {
		t.Errorf("handler returned wrong last item: got %v want %v", lastItem, test.lastItem)
	}
	if rr.Body.String() != test.want {
		t.Errorf("handler returned unexpected body: got '%v' want '%v'", rr.Body.String(), test.want)
	} */
}

func TestBulkIncHandler(t *testing.T) {
	s := getEmptyStore(t)
	bulkAddTestItems(t, s)

	// Can we bulk increment some of the items' attempts?
	body := []byte(`a
b
c
d
e`)
	req, err := http.NewRequest("BULKINCREMENT", "/lists/downloads", bytes.NewBuffer(body))
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	h := &Handler{Store: s}
	handler := http.Handler(h)
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
	expected := "INCREMENTED 5\n"
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
	}

	// If we look for incremented items, are they incremented?
	for _, file := range []string{"a", "b", "c", "d", "e"} {
		attempts, ok, err := s.Get(context.Background(), "downloads", file)
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
		attempts, ok, err := s.Get(context.Background(), "downloads", file)
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

	// What if we bulk increment nothing?
	req, err = http.NewRequest("BULKINCREMENT", "/lists/downloads", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr = httptest.NewRecorder()
	h = &Handler{Store: s}    // XXX: needed?
	handler = http.Handler(h) // XXX: needed?
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
	expected = "INCREMENTED 0\n"
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
	}
}

func TestBulkDelHandler(t *testing.T) {
	s := getEmptyStore(t)
	bulkAddTestItems(t, s)

	// Can we bulk delete some of the items?
	body := []byte(`a
b
c
d
e`)
	req, err := http.NewRequest("BULKDELETE", "/lists/downloads", bytes.NewBuffer(body))
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	h := &Handler{Store: s}
	handler := http.Handler(h)
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Wrong status code: got %v want %v", status, http.StatusOK)
	}
	expected := "DELETED 5\n"
	if rr.Body.String() != expected {
		t.Errorf("Unexpected body: got %v want %v", rr.Body.String(), expected)
	}

	// If we look for the deleted items, are they correctly missing?
	for _, file := range []string{"a", "b", "c", "d", "e"} {
		_, ok, err := s.Get(context.Background(), "downloads", file)
		if err != nil {
			t.Errorf("Error getting item: %v", err)
		}
		if ok {
			t.Errorf("Found item %v that should have been deleted from list.", file)
		}
	}

	// Were other items left alone?
	for _, file := range []string{"f", "g"} {
		attempts, ok, err := s.Get(context.Background(), "downloads", file)
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

	// What if we bulk delete nothing?
	req, err = http.NewRequest("BULKDELETE", "/lists/downloads", bytes.NewBuffer(body))
	if err != nil {
		t.Fatal(err)
	}
	rr = httptest.NewRecorder()
	h = &Handler{Store: getEmptyStore(t)} // XXX needed?
	handler = http.Handler(h)             // XXX needed?
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Wrong status code: got %v want %v", status, http.StatusOK)
	}
	expected = "DELETED 0\n"
	if rr.Body.String() != expected {
		t.Errorf("Unexpected body: got %v want %v", rr.Body.String(), expected)
	}
}
