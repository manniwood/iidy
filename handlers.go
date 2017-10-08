package iidy

import (
	"fmt"
	"net/http"
)

type Env struct {
	Store *MemStore
}

type Handler struct {
	Env *Env
	H   func(e *Env, w http.ResponseWriter, r *http.Request)
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.H(h.Env, w, r)
}

// OK, but how do I get a reference to an instance of
// the datastore?
func HelloWorldHandler(e *Env, w http.ResponseWriter, r *http.Request) {
	// e now has our env
	fmt.Fprint(w, "Hello World\n")
}
