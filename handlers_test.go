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

	_, ok, _ := env.Store.Get("downloads", "linux.tar.gz")
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
	expected := []ListItem{
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

	listItems, err := env.Store.BulkGet("downloads", "", 3)
	if err != nil {
		t.Errorf("Error fetching items: %v", err)
	}
	if !ItemSlicesAreEqual(expected, listItems) {
		t.Errorf("Expected %v; got %v", expected, listItems)
	}
}

func TestBulkGetHandler(t *testing.T) {
	s := getEmptyStore(t)
	files := []string{"a", "b", "c", "d", "e", "f", "g"}
	err := s.BulkAdd("downloads", files)
	if err != nil {
		t.Errorf("Error bulk inserting: %w", err)
	}

	var tests = []struct {
		startKey string
		want     string
	}{
		{"", "a 0\nb 0\n"},
		{"b", "c 0\nd 0\n"},
		{"d", "e 0\nf 0\n"},
		{"f", "g 0\n"},
	}
	for _, test := range tests {
		req, err := http.NewRequest("BULKGET", "/lists/downloads", nil)
		if err != nil {
			t.Fatal(err)
		}
		if test.startKey != "" {
			req.Header.Set("X-IIDY-Start-Key", test.startKey)
		}
		req.Header.Set("X-IIDY-Count", "2")

		rr := httptest.NewRecorder()
		env := &Env{Store: s}
		handler := http.Handler(Handler{Env: env, H: ListHandler})

		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}
		if rr.Body.String() != test.want {
			t.Errorf("handler returned unexpected body: got '%v' want '%v'", rr.Body.String(), test.want)
		}
	}
}
