package iidy

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/manniwood/iidy/pgstore"
)

// TODO: any json response bodies should probably be parsed into
// structs and deep equalled.

func TestPostHandler(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "/iidy/v1/lists/downloads/kernel.tar.gz", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	h := &Handler{Store: getEmptyStore(t)}
	handler := http.Handler(h)
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusCreated)
	}
	expected := "ADDED 1\n"
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
	}

	// Did we really add the item?
	_, ok, err := h.Store.GetOne(context.Background(), "downloads", "kernel.tar.gz")
	if err != nil {
		t.Errorf("Error getting item: %v", err)
	}
	if !ok {
		t.Error("Did not properly get item from list.")
	}
}

func TestNonExistentMethod(t *testing.T) {
	req, err := http.NewRequest("BLARG", "/iidy/v1/lists/downloads/kernel.tar.gz", nil)
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
	req, err := http.NewRequest("GET", "/iidy/v1/lists/downloads/kernel.tar.gz", nil)
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
	req, err := http.NewRequest("GET", "/iidy/v1/lists/downloads/i_do_not_exist.tar.gz", nil)
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
	req, err = http.NewRequest("GET", "/iidy/v1/lists/i_do_not_exist/kernel.tar.gz", nil)
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
	req, err := http.NewRequest("POST", "/iidy/v1/lists/downloads/kernel.tar.gz?action=increment", nil)
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
	req, err = http.NewRequest("GET", "/iidy/v1/lists/downloads/kernel.tar.gz", nil)
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
	req, err = http.NewRequest("POST", "/iidy/v1/lists/i_do_not_exist/kernel.tar.gz?action=increment", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr = httptest.NewRecorder()
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
	req, err := http.NewRequest("DELETE", "/iidy/v1/lists/downloads/kernel.tar.gz", nil)
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
	req, err = http.NewRequest("GET", "/iidy/v1/lists/downloads/kernel.tar.gz", nil)
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
	req, err = http.NewRequest("DELETE", "/iidy/v1/lists/downloads/kernel.tar.gz", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
	expected = "DELETED 0\n"
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
	}

}

func TestBatchPostHandler(t *testing.T) {
	var tests = []struct {
		mime           string
		body           []byte
		expectAfterAdd string
		expected       []pgstore.ListEntry
	}{
		{
			mime: "text/plain",
			body: []byte(`kernel.tar.gz
vim.tar.gz
robots.txt`),
			expectAfterAdd: "ADDED 3\n",
			// remember, these come back in alphabetical order
			expected: []pgstore.ListEntry{
				{"kernel.tar.gz", 0},
				{"robots.txt", 0},
				{"vim.tar.gz", 0},
			},
		},
		{
			mime: "application/json",
			body: []byte(`{ "items": ["kernel.tar.gz", "vim.tar.gz", "robots.txt"] }`),
			expectAfterAdd: `{"added":3}
`,
			// remember, these come back in alphabetical order
			expected: []pgstore.ListEntry{
				{"kernel.tar.gz", 0},
				{"robots.txt", 0},
				{"vim.tar.gz", 0},
			},
		},
		{
			mime:           "text/plain",
			body:           nil,
			expectAfterAdd: "ADDED 0\n",
			// remember, these come back in alphabetical order
			expected: []pgstore.ListEntry{},
		},
		{
			mime: "application/json",
			body: nil,
			expectAfterAdd: `{"added":0}
`,
			// remember, these come back in alphabetical order
			expected: []pgstore.ListEntry{},
		},
	}

	for _, test := range tests {
		h := &Handler{Store: getEmptyStore(t)}
		// First, clear the store.
		h.Store.Nuke(context.Background())

		req, err := http.NewRequest("POST", "/iidy/v1/bulk/lists/downloads", bytes.NewBuffer(test.body))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", test.mime)
		rr := httptest.NewRecorder()
		handler := http.Handler(h)
		handler.ServeHTTP(rr, req)
		if status := rr.Code; status != http.StatusCreated {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusCreated)
		}
		if rr.Body.String() != test.expectAfterAdd {
			t.Errorf(`Unexpected body: got "%v" want "%v"`, rr.Body.String(), test.expectAfterAdd)
		}

		// What if we bulk get what we just bulk put?
		listEntries, err := h.Store.GetBatch(context.Background(), "downloads", "", 3)
		if err != nil {
			t.Errorf("Error fetching items: %v", err)
		}
		if !reflect.DeepEqual(test.expected, listEntries) {
			t.Errorf("Expected %v; got %v", test.expected, listEntries)
		}
	}
}

func TestBatchGetHandler(t *testing.T) {
	// Order of these tests matters. We set up state and go through in order.
	var tests = []struct {
		afterItem string
		want      string
		wantJSON  string
		lastItem  string
	}{
		{
			afterItem: "",
			want:      "a 0\nb 0\n",
			wantJSON: `{"listentries":[{"item":"a","attempts":0},{"item":"b","attempts":0}]}
`,
			lastItem: "b",
		},
		{
			afterItem: "b",
			want:      "c 0\nd 0\n",
			wantJSON: `{"listentries":[{"item":"c","attempts":0},{"item":"d","attempts":0}]}
`,
			lastItem: "d",
		},
		{
			afterItem: "d",
			want:      "e 0\nf 0\n",
			wantJSON: `{"listentries":[{"item":"e","attempts":0},{"item":"f","attempts":0}]}
`,
			lastItem: "f",
		},
		{
			afterItem: "f",
			want:      "g 0\n",
			wantJSON: `{"listentries":[{"item":"g","attempts":0}]}
`,
			lastItem: "g",
		},
	}

	s := getEmptyStore(t)
	bulkAddTestItems(t, s)

	for _, mime := range []string{"text/plain", "application/json"} {
		for _, test := range tests {
			var want string
			if mime == "text/plain" {
				want = test.want
			} else {
				want = test.wantJSON
			}

			url := "/iidy/v1/bulk/lists/downloads?count=2"
			if test.afterItem != "" {
				url += "&after_id="
				url += test.afterItem
			}
			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Content-Type", mime)
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
			if rr.Body.String() != want {
				t.Errorf("handler returned unexpected body: got '%v' want '%v'", rr.Body.String(), want)
			}
		}
	}
}

func TestBatchGetHandlerError(t *testing.T) {
	// What if we bulk get from a list that doesn't exist?
	req, err := http.NewRequest("GET", "/iidy/v1/bulk/lists/i_do_not_exist?count=2", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	s := getEmptyStore(t)
	h := &Handler{Store: s}
	handler := http.Handler(h)
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
}

func TestBatchIncHandler(t *testing.T) {
	var tests = []struct {
		name     string
		mime     string
		body     []byte
		expected string
	}{
		{
			name: "text",
			mime: "text/plain",
			body: []byte(`a
b
c
d
e`),
			expected: "INCREMENTED 5\n",
		},
		{
			name: "JSON",
			mime: "application/json",
			body: []byte(`{ "items": ["a", "b", "c", "d", "e"] }`),
			expected: `{"incremented":5}
`,
		},
	}
	for _, test := range tests {
		s := getEmptyStore(t)
		bulkAddTestItems(t, s)

		// Can we bulk increment some of the items' attempts?
		req, err := http.NewRequest("POST", "/iidy/v1/bulk/lists/downloads?action=increment", bytes.NewBuffer(test.body))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", test.mime)
		rr := httptest.NewRecorder()
		h := &Handler{Store: s}
		handler := http.Handler(h)
		handler.ServeHTTP(rr, req)
		if status := rr.Code; status != http.StatusOK {
			t.Errorf("%s: handler returned wrong status code: got %v want %v", test.name, status, http.StatusOK)
		}
		if rr.Body.String() != test.expected {
			t.Errorf("%s: handler returned unexpected body: got %v want %v", test.name, rr.Body.String(), test.expected)
		}

		// If we look for incremented items, are they incremented?
		for _, file := range []string{"a", "b", "c", "d", "e"} {
			attempts, ok, err := s.GetOne(context.Background(), "downloads", file)
			if err != nil {
				t.Errorf("%s: Error getting item: %v", test.name, err)
			}
			if !ok {
				t.Errorf("%s: Did not properly get item %v from list.", test.name, file)
			}
			if attempts != 1 {
				t.Errorf("%s: Did not properly increment item %v.", test.name, file)
			}
		}

		// What about non-incremented items? Were they left alone?
		for _, file := range []string{"f", "g"} {
			attempts, ok, err := s.GetOne(context.Background(), "downloads", file)
			if err != nil {
				t.Errorf("%s: Error getting item: %v", test.name, err)
			}
			if !ok {
				t.Errorf("%s: Did not properly get item %v from list.", test.name, file)
			}
			if attempts != 0 {
				t.Errorf("%s: Item %v is incorrectly incremented.", test.name, file)
			}
		}
	}
}

func TestBatchIncHandlerError(t *testing.T) {
	var tests = []struct {
		name     string
		mime     string
		expected string
	}{
		{
			name:     "text",
			mime:     "text/plain",
			expected: "INCREMENTED 0\n",
		},
		{
			name: "JSON",
			mime: "application/json",
			expected: `{"incremented":0}
`,
		},
	}
	for _, test := range tests {
		// What if we bulk increment nothing?
		req, err := http.NewRequest(http.MethodPost, "/iidy/v1/bulk/lists/downloads?action=increment", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", test.mime)
		rr := httptest.NewRecorder()
		s := getEmptyStore(t)
		h := &Handler{Store: s}
		handler := http.Handler(h)
		handler.ServeHTTP(rr, req)
		if status := rr.Code; status != http.StatusOK {
			t.Errorf("%s: handler returned wrong status code: got %v want %v", test.name, status, http.StatusOK)
		}
		if rr.Body.String() != test.expected {
			t.Errorf("%s: handler returned unexpected body: got %v want %v", test.name, rr.Body.String(), test.expected)
		}
	}
}

func TestBatchDelHandler(t *testing.T) {
	var tests = []struct {
		name     string
		mime     string
		body     []byte
		expected string
	}{
		{
			name: "text",
			mime: "text/plain",
			body: []byte(`a
b
c
d
e`),
			expected: "DELETED 5\n",
		},
		{
			name: "JSON",
			mime: "application/json",
			body: []byte(`{ "items": ["a", "b", "c", "d", "e"] }`),
			expected: `{"deleted":5}
`,
		},
	}
	for _, test := range tests {
		s := getEmptyStore(t)
		bulkAddTestItems(t, s)

		req, err := http.NewRequest(http.MethodDelete, "/iidy/v1/bulk/lists/downloads", bytes.NewBuffer(test.body))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", test.mime)
		rr := httptest.NewRecorder()
		h := &Handler{Store: s}
		handler := http.Handler(h)
		handler.ServeHTTP(rr, req)
		if status := rr.Code; status != http.StatusOK {
			t.Errorf("%s: handler returned wrong status code: got %v want %v", test.name, status, http.StatusOK)
		}
		if rr.Body.String() != test.expected {
			t.Errorf("%s: handler returned unexpected body: got %v want %v", test.name, rr.Body.String(), test.expected)
		}

		// If we look for the deleted items, are they correctly missing?
		for _, file := range []string{"a", "b", "c", "d", "e"} {
			_, ok, err := s.GetOne(context.Background(), "downloads", file)
			if err != nil {
				t.Errorf("%s: Error getting item: %v", test.name, err)
			}
			if ok {
				t.Errorf("%s: Found item %v that should have been deleted from list.", test.name, file)
			}
		}

		// Were other items left alone?
		for _, file := range []string{"f", "g"} {
			attempts, ok, err := s.GetOne(context.Background(), "downloads", file)
			if err != nil {
				t.Errorf("%s: Error getting item: %v", test.name, err)
			}
			if !ok {
				t.Errorf("%s: Item %v should not have been deleted from list.", test.name, file)
			}
			if attempts != 0 {
				t.Errorf("%s: Item %v is incorrectly incremented.", test.name, file)
			}
		}
	}
}

func TestBatchDelHandlerError(t *testing.T) {
	var tests = []struct {
		name     string
		mime     string
		expected string
	}{
		{
			name:     "text",
			mime:     "text/plain",
			expected: "DELETED 0\n",
		},
		{
			name: "JSON",
			mime: "application/json",
			expected: `{"deleted":0}
`,
		},
	}
	for _, test := range tests {
		s := getEmptyStore(t)
		h := &Handler{Store: s}
		// What if we bulk delete nothing?
		// First, clear the store.
		h.Store.Nuke(context.Background())
		req, err := http.NewRequest(http.MethodDelete, "/iidy/v1/bulk/lists/downloads", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", test.mime)
		rr := httptest.NewRecorder()
		handler := http.Handler(h)
		handler.ServeHTTP(rr, req)
		if status := rr.Code; status != http.StatusOK {
			t.Errorf("%s: Wrong status code: got %v want %v", test.name, status, http.StatusOK)
		}
		if rr.Body.String() != test.expected {
			t.Errorf("%s: Unexpected body: got %v want %v", test.name, rr.Body.String(), test.expected)
		}
	}
}

func getEmptyStore(t *testing.T) *pgstore.PgStore {
	p, err := pgstore.NewPgStore("")
	if err != nil {
		t.Errorf("Error instantiating PgStore: %v", err)
	}
	p.Nuke(context.Background())
	return p
}

// Our tests add this test item over and over,
// so here it is.
func addSingleStartingItem(t *testing.T, s *pgstore.PgStore) {
	count, err := s.InsertOne(context.Background(), "downloads", "kernel.tar.gz")
	if err != nil {
		t.Errorf("Error adding item: %v", err)
	}
	if count != 1 {
		t.Error("Did not properly add item to list.")
	}
}

// These items are expected to be in the db at the start
// of the next few bulk tests.
func bulkAddTestItems(t *testing.T, s *pgstore.PgStore) {
	// Batch add a bunch of test items.
	files := []string{"a", "b", "c", "d", "e", "f", "g"}
	count, err := s.InsertBatch(context.Background(), "downloads", files)
	if err != nil {
		t.Errorf("Error bulk inserting: %v", err)
	}
	if count != 7 {
		t.Errorf("Batch added wrong number of items. Expected 5, got %v", count)
	}
}
