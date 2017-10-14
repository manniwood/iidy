package iidy

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPutHandler(t *testing.T) {
	req, err := http.NewRequest("PUT", "/lists/downloads/linux.tar.gz", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	env := &Env{Store: getEmptyStore(t)}
	handler := http.Handler(Handler{Env: env, H: ListHandler})

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	expected := "ADDED: downloads, linux.tar.gz\n"
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
	}

	_, ok, err := env.Store.Get("downloads", "linux.tar.gz")
	if err != nil {
		t.Errorf("Error getting item: %v", err)
	}
	if !ok {
		t.Error("Did not properly get item from list.")
	}
}

func TestGetHandler(t *testing.T) {

	env := &Env{Store: getEmptyStore(t)}
	putSingleStartingValue(t, env)

	// now, get the value
	req, err := http.NewRequest("GET", "/lists/downloads/linux.tar.gz", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.Handler(Handler{Env: env, H: ListHandler})

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	expected := "0\n"
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
	}
}

func TestIncHandler(t *testing.T) {

	env := &Env{Store: getEmptyStore(t)}
	putSingleStartingValue(t, env)

	// now, increment the value
	req, err := http.NewRequest("INCREMENT", "/lists/downloads/linux.tar.gz", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.Handler(Handler{Env: env, H: ListHandler})

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	expected := "INCREMENTED: downloads, linux.tar.gz\n"
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
	}

	// Now the new value should be fetchable with GET
	req, err = http.NewRequest("GET", "/lists/downloads/linux.tar.gz", nil)
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
}

func TestDelHandler(t *testing.T) {

	env := &Env{Store: getEmptyStore(t)}
	putSingleStartingValue(t, env)

	// Now the value should be deletable with DELETE
	req, err := http.NewRequest("DELETE", "/lists/downloads/linux.tar.gz", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.Handler(Handler{Env: env, H: ListHandler})

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	expected := "DELETED: downloads, linux.tar.gz\n"
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
	}

	// Now the deleted value should not be fetchable with GET
	req, err = http.NewRequest("GET", "/lists/downloads/linux.tar.gz", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// NOTE to test maintainers: a trailing newline is added for us by http.Error
	expected = "Not found.\n"
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got '%v' want '%v'", rr.Body.String(), expected)
	}
}

func putSingleStartingValue(t *testing.T, env *Env) {
	req, err := http.NewRequest("PUT", "/lists/downloads/linux.tar.gz", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.Handler(Handler{Env: env, H: ListHandler})

	handler.ServeHTTP(rr, req)
}

func TestBulkPutHandler(t *testing.T) {
	body := []byte(`linux.tar.gz
vim.tar.gz
robots.txt`)
	// remember, these come back in alphabetical order
	expected := []ListEntry{
		{"linux.tar.gz", 0},
		{"robots.txt", 0},
		{"vim.tar.gz", 0},
	}
	req, err := http.NewRequest("BULKPUT", "/lists/downloads", bytes.NewBuffer(body))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	env := &Env{Store: getEmptyStore(t)}
	handler := http.Handler(Handler{Env: env, H: ListHandler})

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	listEntries, err := env.Store.BulkGet("downloads", "", 3)
	if err != nil {
		t.Errorf("Error fetching items: %v", err)
	}
	if !ListEntrySlicesAreEqual(expected, listEntries) {
		t.Errorf("Expected %v; got %v", expected, listEntries)
	}
}

func TestBulkGetHandler(t *testing.T) {
	s := getEmptyStore(t)
	files := []string{"a", "b", "c", "d", "e", "f", "g"}
	err := s.BulkAdd("downloads", files)
	if err != nil {
		t.Errorf("Error bulk inserting: %v", err)
	}

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
		env := &Env{Store: s}
		handler := http.Handler(Handler{Env: env, H: ListHandler})

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
}

func TestBulkIncHandler(t *testing.T) {
	s := getEmptyStore(t)
	files := []string{"a", "b", "c", "d", "e", "f", "g"}
	err := s.BulkAdd("downloads", files)
	if err != nil {
		t.Errorf("Error bulk inserting: %v", err)
	}
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
	env := &Env{Store: s}
	handler := http.Handler(Handler{Env: env, H: ListHandler})

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	expected := "INCREMENTED 5\n"
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
	}

	for _, file := range []string{"a", "b", "c", "d", "e"} {
		attempts, ok, err := s.Get("downloads", file)
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
	for _, file := range []string{"f", "g"} {
		attempts, ok, err := s.Get("downloads", file)
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
}

func TestBulkDelHandler(t *testing.T) {
	s := getEmptyStore(t)
	files := []string{"a", "b", "c", "d", "e", "f", "g"}
	err := s.BulkAdd("downloads", files)
	if err != nil {
		t.Errorf("Error bulk inserting: %v", err)
	}
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
	env := &Env{Store: s}
	handler := http.Handler(Handler{Env: env, H: ListHandler})

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Wrong status code: got %v want %v", status, http.StatusOK)
	}

	expected := "DELETED 5\n"
	if rr.Body.String() != expected {
		t.Errorf("Unexpected body: got %v want %v", rr.Body.String(), expected)
	}
	for _, file := range []string{"a", "b", "c", "d", "e"} {
		_, ok, err := s.Get("downloads", file)
		if err != nil {
			t.Errorf("Error getting item: %v", err)
		}
		if ok {
			t.Errorf("Found item %v that should have been deleted from list.", file)
		}
	}
	for _, file := range []string{"f", "g"} {
		attempts, ok, err := s.Get("downloads", file)
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
}
