package iidy

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHelloWorldHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/helloworld", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	env := &Env{Store: NewMemStore()}
	handler := http.Handler(Handler{Env: env, H: HelloWorldHandler})

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	expected := "Hello World\n"
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
	}
}

func TestListHandler(t *testing.T) {
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
