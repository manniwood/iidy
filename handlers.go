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
	Store Store
}

type Handler struct {
	Env *Env
	H   func(e *Env, w http.ResponseWriter, r *http.Request)
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.H(h.Env, w, r)
}

func ListHandler(e *Env, w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "PUT":
		parseNames(e, w, r, PutHandler)
	case "GET":
		parseNames(e, w, r, GetHandler)
	case "INC":
		parseNames(e, w, r, IncHandler)
	case "DEL":
		parseNames(e, w, r, DelHandler)
	default:
		http.Error(w, "Unknown method.", http.StatusBadRequest)
	}
}

type internalHandler func(e *Env, w http.ResponseWriter, r *http.Request, listName string, itemName string)

func parseNames(e *Env, w http.ResponseWriter, r *http.Request, h internalHandler) {
	urlParts := strings.Split(r.URL.Path, "/")
	if len(urlParts) != 4 {
		http.Error(w, "Bad request; needs to look like /lists/<listname>/<itemname>", http.StatusBadRequest)
		return
	}
	listName := urlParts[2]
	itemName := urlParts[3]
	h(e, w, r, listName, itemName)
}

func PutHandler(e *Env, w http.ResponseWriter, r *http.Request, listName string, itemName string) {
	err := e.Store.Add(listName, itemName)
	if err != nil {
		errStr := fmt.Sprintf("Error processing request; %v", err)
		http.Error(w, errStr, http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "ADDED: %s, %s\n", listName, itemName)
}

func IncHandler(e *Env, w http.ResponseWriter, r *http.Request, listName string, itemName string) {
	err := e.Store.Inc(listName, itemName)
	if err != nil {
		errStr := fmt.Sprintf("Error processing request; %v", err)
		http.Error(w, errStr, http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "INCREMENTED: %s, %s\n", listName, itemName)
}

func DelHandler(e *Env, w http.ResponseWriter, r *http.Request, listName string, itemName string) {
	err := e.Store.Del(listName, itemName)
	if err != nil {
		errStr := fmt.Sprintf("Error processing request; %v", err)
		http.Error(w, errStr, http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "DELETED: %s, %s\n", listName, itemName)
}

func GetHandler(e *Env, w http.ResponseWriter, r *http.Request, listName string, itemName string) {
	attempts, ok, err := e.Store.Get(listName, itemName)
	if err != nil {
		errStr := fmt.Sprintf("Error processing request; %v", err)
		http.Error(w, errStr, http.StatusInternalServerError)
		return
	}
	if !ok {
		http.Error(w, "Not found.", http.StatusNotFound)
		return
	}
	fmt.Fprintf(w, "%d\n", attempts)
}
