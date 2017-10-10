package iidy

import (
	"fmt"
	"net/http"
)

// The whole way of setting up these structs
// and the handler so that I can provide access
// to the Store is inspired by
// https://elithrar.github.io/article/http-handler-error-handling-revisited/

type Env struct {
	Store Store
}

type Handler struct {
	Env *Env
	H   func(e *Env, w http.ResponseWriter, r *http.Request)
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.H(h.Env, w, r)
}

func HelloWorldHandler(e *Env, w http.ResponseWriter, r *http.Request) {
	// e now has our env
	fmt.Fprint(w, "Hello World\n")
}
