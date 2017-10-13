package iidy

import (
	"fmt"
	"net/http"
	"strings"
)

// The whole way of setting up these structs
// and the handler so that I can provide access
// to the Store is inspired by
// https://elithrar.github.io/article/http-handler-error-handling-revisited/

type Env struct {
	Store *PgStore
}

type Handler struct {
	Env *Env
	H   func(e *Env, w http.ResponseWriter, r *http.Request)
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.H(h.Env, w, r)
}

func ListHandler(e *Env, w http.ResponseWriter, r *http.Request) {
	urlParts := strings.Split(r.URL.Path, "/")
	if len(urlParts) != 4 {
		http.Error(w, "Bad request; needs to look like /lists/<listname>/<itemname>", http.StatusBadRequest)
		return
	}
	listName := urlParts[2]
	itemName := urlParts[3]

	switch r.Method {
	case "PUT":
		PutHandler(e, w, r, listName, itemName)
	case "GET":
		GetHandler(e, w, r, listName, itemName)
	case "INCREMENT":
		IncHandler(e, w, r, listName, itemName)
	case "DELETE":
		DelHandler(e, w, r, listName, itemName)
	default:
		http.Error(w, "Unknown method.", http.StatusBadRequest)
	}
}

func PutHandler(e *Env, w http.ResponseWriter, r *http.Request, listName string, itemName string) {
	err := e.Store.Add(listName, itemName)
	if err != nil {
		errStr := fmt.Sprintf("Error trying to add list item: %v", err)
		http.Error(w, errStr, http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "ADDED: %s, %s\n", listName, itemName)
}

func IncHandler(e *Env, w http.ResponseWriter, r *http.Request, listName string, itemName string) {
	err := e.Store.Inc(listName, itemName)
	if err != nil {
		errStr := fmt.Sprintf("Error trying to increment list item: %v", err)
		http.Error(w, errStr, http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "INCREMENTED: %s, %s\n", listName, itemName)
}

func DelHandler(e *Env, w http.ResponseWriter, r *http.Request, listName string, itemName string) {
	err := e.Store.Del(listName, itemName)
	if err != nil {
		errStr := fmt.Sprintf("Error trying to delete list item: %v", err)
		http.Error(w, errStr, http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "DELETED: %s, %s\n", listName, itemName)
}

func GetHandler(e *Env, w http.ResponseWriter, r *http.Request, listName string, itemName string) {
	attempts, ok, err := e.Store.Get(listName, itemName)
	if err != nil {
		errStr := fmt.Sprintf("Error trying to get list item: %v", err)
		http.Error(w, errStr, http.StatusInternalServerError)
		return
	}
	if !ok {
		http.Error(w, "Not found.", http.StatusNotFound)
		return
	}
	fmt.Fprintf(w, "%d\n", attempts)
}
