package iidy

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
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
	var listName string
	var itemName string
	switch r.Method {
	case "PUT", "GET", "INCREMENT", "DELETE":
		if len(urlParts) != 4 {
			http.Error(w, "Bad request; needs to look like /lists/<listname>/<itemname>", http.StatusBadRequest)
			return
		}
		listName = urlParts[2]
		itemName = urlParts[3]
	case "BULKPUT", "BULKGET", "BULKINCREMENT", "BULKDELETE":
		if len(urlParts) != 3 {
			http.Error(w, "Bad request; needs to look like /lists/<listname>", http.StatusBadRequest)
			return
		}
		listName = urlParts[2]
	default:
		http.Error(w, "Unknown method.", http.StatusBadRequest)
	}

	switch r.Method {
	case "PUT":
		PutHandler(e, w, r, listName, itemName)
	case "GET":
		GetHandler(e, w, r, listName, itemName)
	case "INCREMENT":
		IncHandler(e, w, r, listName, itemName)
	case "DELETE":
		DelHandler(e, w, r, listName, itemName)
	case "BULKPUT":
		BulkPutHandler(e, w, r, listName)
	case "BULKGET":
		BulkGetHandler(e, w, r, listName)
	case "BULKINCREMENT":
		BulkIncHandler(e, w, r, listName)
	case "BULKDELETE":
		BulkDelHandler(e, w, r, listName)
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

func BulkPutHandler(e *Env, w http.ResponseWriter, r *http.Request, listName string) {
	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		errStr := fmt.Sprintf("Error reading body: %v", err)
		http.Error(w, errStr, http.StatusBadRequest)
		return
	}
	// TODO: trim trailing newlines from bodyBytes first.
	itemNames := strings.Split(string(bodyBytes[:]), "\n")

	err = e.Store.BulkAdd(listName, itemNames)
	if err != nil {
		errStr := fmt.Sprintf("Error trying to add list items: %v", err)
		http.Error(w, errStr, http.StatusInternalServerError)
		return
	}
	fmt.Fprint(w, "ADDED")
}

func BulkGetHandler(e *Env, w http.ResponseWriter, r *http.Request, listName string) {
	startID := r.Header.Get("X-IIDY-Start-Key")
	countStr := r.Header.Get("X-IIDY-Count")
	if countStr == "" {
		http.Error(w, "Header not found: X-IIDY-Count", http.StatusBadRequest)
		return
	}
	count, err := strconv.Atoi(countStr)
	if err != nil {
		errStr := fmt.Sprintf("For header X-IIDY-Count, %v is not a number: %v", countStr, err)
		http.Error(w, errStr, http.StatusInternalServerError)
		return
	}
	listItems, err := e.Store.BulkGet(listName, startID, count)
	for _, listItem := range listItems {
		fmt.Fprintf(w, "%s %d\n", listItem.Item, listItem.Attempts)
	}
}

func BulkIncHandler(e *Env, w http.ResponseWriter, r *http.Request, listName string) {
	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		errStr := fmt.Sprintf("Error reading body: %v", err)
		http.Error(w, errStr, http.StatusBadRequest)
		return
	}
	// TODO: trim trailing newlines from bodyBytes first.
	itemNames := strings.Split(string(bodyBytes[:]), "\n")

	count, err := e.Store.BulkInc(listName, itemNames)
	if err != nil {
		errStr := fmt.Sprintf("Error trying to increment list items: %v", err)
		http.Error(w, errStr, http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "INCREMENTED %d\n", count)
}

func BulkDelHandler(e *Env, w http.ResponseWriter, r *http.Request, listName string) {
	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		errStr := fmt.Sprintf("Error reading body: %v", err)
		http.Error(w, errStr, http.StatusBadRequest)
		return
	}
	// TODO: trim trailing newlines from bodyBytes first.
	itemNames := strings.Split(string(bodyBytes[:]), "\n")

	count, err := e.Store.BulkDel(listName, itemNames)
	if err != nil {
		errStr := fmt.Sprintf("Error trying to delete list items: %v", err)
		http.Error(w, errStr, http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "DELETED %d\n", count)
}
