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

// Env is a global struct that's only meant to be instantiated
// once, like a singleton, and passed in to ListHandler so that
// our handlers have access to the data store.
type Env struct {
	Store *PgStore
}

// Handler is a struct that takes a pointer to and Env struct
// and then hands that in to a function that satisfies the
// http.Handler interface.
type Handler struct {
	Env *Env
	H   func(e *Env, w http.ResponseWriter, r *http.Request)
}

// ServeHTTP satisfies the http.Handler interface, but
// actually calls our Handler struct's H function, which
// has access to our Env "singleton" which holds a pointer
// to our data store.
func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.H(h.Env, w, r)
}

// ListHandler is expected to handle all traffic to "/lists/".
// It parses out the list and item names from the URL and then
// delegates to more specific handlers.
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

// PutHandler adds an item to a list. If the list does not already
// exist, it will be created.
func PutHandler(e *Env, w http.ResponseWriter, r *http.Request, list string, item string) {
	count, err := e.Store.Add(list, item)
	if err != nil {
		errStr := fmt.Sprintf("Error trying to add list item: %v", err)
		http.Error(w, errStr, http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "ADDED %d\n", count)
}

// IncHandler increments an item in a list. The returned
// body text says the number of items found and incremented (1 or 0).
func IncHandler(e *Env, w http.ResponseWriter, r *http.Request, list string, item string) {
	count, err := e.Store.Inc(list, item)
	if err != nil {
		errStr := fmt.Sprintf("Error trying to increment list item: %v", err)
		http.Error(w, errStr, http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "INCREMENTED %d\n", count)
}

// DelHandler deletes an item from a list. The returned
// body text says the number of items found and deleted (1 or 0).
func DelHandler(e *Env, w http.ResponseWriter, r *http.Request, list string, item string) {
	count, err := e.Store.Del(list, item)
	if err != nil {
		errStr := fmt.Sprintf("Error trying to delete list item: %v", err)
		http.Error(w, errStr, http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "DELETED %d\n", count)
}

// GetHandler returns the number of attempts that were made to
// complete an item in a list. When a list or list item
// is missing, the client will get a 404 instead.
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

// BulkPutHandler adds all of the items in the request body
// (item names separated by newlines) to the specified
// list, and sets their completion attempt counts to 0.
// The response contains the number of items successfully
// inserted, generally len(items) or 0.
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

// BulkGetHandler requires the "X-IIDY-Count" header, and takes
// an optional "X-IIDY-After-Item" header. It returns a response body of
// list items; each list item is followed by a space and the number of
// attempts to complete that list item. Each list item / attempt count
// pair is separated by a newline. "X-IIDY-Count" determines how many
// items are returned (from the sorted list). "X-IIDY-After-Item" determines
// the offset in the list; when set to the empty string, we start at the
// beginning of the list; when set to an item (generally the last item
// from a previous call to this handler) we start after that item in
// the list.
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

// BulkIncHandler increments all of the items in the request body
// (item names separated by newlines) in the specified
// list. The response contains the number of items successfully
// incremented, generally len(items) or 0.
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

// BulkDelHandler deletes all of the items in the request body
// (item names separated by newlines) from the specified
// list. The response contains the number of items successfully
// deleted, generally len(items) or 0.
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
