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
	// We always deal in plain text, so may as well be explicit about it.
	w.Header().Set("Content-Type", "text/plain")
	urlParts := strings.Split(r.URL.Path, "/")
	var list string
	var item string
	switch r.Method {
	case "PUT", "GET", "INCREMENT", "DELETE":
		if len(urlParts) != 4 {
			http.Error(w, "Bad request; needs to look like /lists/<listname>/<itemname>", http.StatusBadRequest)
			return
		}
		list = urlParts[2]
		item = urlParts[3]
	case "BULKPUT", "BULKGET", "BULKINCREMENT", "BULKDELETE":
		if len(urlParts) != 3 {
			http.Error(w, "Bad request; needs to look like /lists/<listname>", http.StatusBadRequest)
			return
		}
		list = urlParts[2]
	default:
		http.Error(w, "Unknown method.", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case "PUT":
		PutHandler(e, w, r, list, item)
	case "GET":
		GetHandler(e, w, r, list, item)
	case "INCREMENT":
		IncHandler(e, w, r, list, item)
	case "DELETE":
		DelHandler(e, w, r, list, item)
	case "BULKPUT":
		BulkPutHandler(e, w, r, list)
	case "BULKGET":
		BulkGetHandler(e, w, r, list)
	case "BULKINCREMENT":
		BulkIncHandler(e, w, r, list)
	case "BULKDELETE":
		BulkDelHandler(e, w, r, list)
	default:
		http.Error(w, "Unknown method.", http.StatusBadRequest)
	}
}

func PutHandler(e *Env, w http.ResponseWriter, r *http.Request, list string, item string) {
	count, err := e.Store.Add(list, item)
	if err != nil {
		errStr := fmt.Sprintf("Error trying to add list item: %v", err)
		http.Error(w, errStr, http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "ADDED %d\n", count)
}

func IncHandler(e *Env, w http.ResponseWriter, r *http.Request, list string, item string) {
	count, err := e.Store.Inc(list, item)
	if err != nil {
		errStr := fmt.Sprintf("Error trying to increment list item: %v", err)
		http.Error(w, errStr, http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "INCREMENTED %d\n", count)
}

func DelHandler(e *Env, w http.ResponseWriter, r *http.Request, list string, item string) {
	count, err := e.Store.Del(list, item)
	if err != nil {
		errStr := fmt.Sprintf("Error trying to delete list item: %v", err)
		http.Error(w, errStr, http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "DELETED %d\n", count)
}

func GetHandler(e *Env, w http.ResponseWriter, r *http.Request, list string, item string) {
	attempts, ok, err := e.Store.Get(list, item)
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

func getScrubbedLines(bodyBytes []byte) []string {
	bodyString := string(bodyBytes[:])
	// be nice and trim leading and trailing space from body first.
	bodyString = strings.TrimSpace(bodyString)
	return strings.Split(bodyString, "\n")
}

func BulkPutHandler(e *Env, w http.ResponseWriter, r *http.Request, list string) {
	if r.Body == nil {
		fmt.Fprintf(w, "ADDED 0\n")
		return
	}
	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		errStr := fmt.Sprintf("Error reading body: %v", err)
		http.Error(w, errStr, http.StatusBadRequest)
		return
	}
	items := getScrubbedLines(bodyBytes)

	count, err := e.Store.BulkAdd(list, items)
	if err != nil {
		errStr := fmt.Sprintf("Error trying to add list items: %v", err)
		http.Error(w, errStr, http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "ADDED %d\n", count)
}

func BulkGetHandler(e *Env, w http.ResponseWriter, r *http.Request, list string) {
	startID := r.Header.Get("X-IIDY-After-Item")
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
	if count == 0 {
		return
	}
	listEntries, err := e.Store.BulkGet(list, startID, count)
	if len(listEntries) == 0 {
		// Nothing found, so we are done!
		return
	}
	// Although the client can parse out the last item from the body,
	// as a convenience, also provide the last item in a header.
	w.Header().Set("X-IIDY-Last-Item", listEntries[len(listEntries)-1].Item)
	for _, listItem := range listEntries {
		fmt.Fprintf(w, "%s %d\n", listItem.Item, listItem.Attempts)
	}
}

func BulkIncHandler(e *Env, w http.ResponseWriter, r *http.Request, list string) {
	if r.Body == nil {
		fmt.Fprintf(w, "INCREMENTED 0\n")
		return
	}
	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		errStr := fmt.Sprintf("Error reading body: %v", err)
		http.Error(w, errStr, http.StatusBadRequest)
		return
	}
	items := getScrubbedLines(bodyBytes)

	count, err := e.Store.BulkInc(list, items)
	if err != nil {
		errStr := fmt.Sprintf("Error trying to increment list items: %v", err)
		http.Error(w, errStr, http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "INCREMENTED %d\n", count)
}

func BulkDelHandler(e *Env, w http.ResponseWriter, r *http.Request, list string) {
	if r.Body == nil {
		fmt.Fprintf(w, "ADDED 0\n")
		return
	}
	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		errStr := fmt.Sprintf("Error reading body: %v", err)
		http.Error(w, errStr, http.StatusBadRequest)
		return
	}
	items := getScrubbedLines(bodyBytes)

	count, err := e.Store.BulkDel(list, items)
	if err != nil {
		errStr := fmt.Sprintf("Error trying to delete list items: %v", err)
		http.Error(w, errStr, http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "DELETED %d\n", count)
}
