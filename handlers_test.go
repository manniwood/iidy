package iidy

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Do a table test or some such; too much repetition
func TestPutHandler(t *testing.T) {
	req, err := http.NewRequest("PUT", "/lists/downloads/linux.tar.gz", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	env := &Env{Store: NewMemStore()}
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
	// first, put a value (putting is tested above)
	req, err := http.NewRequest("PUT", "/lists/downloads/linux.tar.gz", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	env := &Env{Store: NewMemStore()}
	handler := http.Handler(Handler{Env: env, H: ListHandler})

	handler.ServeHTTP(rr, req)

	// now, get the value
	req, err = http.NewRequest("GET", "/lists/downloads/linux.tar.gz", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()

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
	// first, put a value (putting is tested above)
	req, err := http.NewRequest("PUT", "/lists/downloads/linux.tar.gz", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	env := &Env{Store: NewMemStore()}
	handler := http.Handler(Handler{Env: env, H: ListHandler})

	handler.ServeHTTP(rr, req)

	// now, increment the value
	req, err = http.NewRequest("INC", "/lists/downloads/linux.tar.gz", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()

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
	// first, put a value (putting is tested above)
	req, err := http.NewRequest("PUT", "/lists/downloads/linux.tar.gz", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	env := &Env{Store: NewMemStore()}
	handler := http.Handler(Handler{Env: env, H: ListHandler})

	handler.ServeHTTP(rr, req)

	// Now the value should be deletable with DEL
	req, err = http.NewRequest("DEL", "/lists/downloads/linux.tar.gz", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()

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

	// NOTE that a trailing newline is added for us by http.Error
	expected = "Not found.\n"
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got '%v' want '%v'", rr.Body.String(), expected)
	}
}
