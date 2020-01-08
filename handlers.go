package iidy

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
)

const FinalContentTypeKey string = "final Content-Type"

// HandledContentTypes are the content types handled
// by this service.
var HandledContentTypes = map[string]struct{}{
	"text/plain":       struct{}{},
	"application/json": struct{}{},
}

// ErrorMessage holds an error that can be sent to the client either as
// plain text or JSON.
type ErrorMessage struct {
	Error string `json:"error"`
}

// AddedMessage informs the user how many items were added to a list.
// The message can be formatted either as plain text or JSON.
type AddedMessage struct {
	Added int64 `json:"added"`
}

// IncrementedMessage informs the user how many items were incremented in a list.
// The message can be formatted either as plain text or JSON.
type IncrementedMessage struct {
	Incremented int64 `json:"incremented"`
}

// DeletedMessage informs the user how many items were deleted from a list.
// The message can be formatted either as plain text or JSON.
type DeletedMessage struct {
	Deleted int64 `json:"deleted"`
}

// Handler handles requests to "/lists/". It contains an instance of PgStore,
// so that it has a place to store list data.
type Handler struct {
	Store *PgStore
}

// ServeHTTP satisfies the http.Handler interface. It is expected to handle
// all traffic to "/lists/". It parses out the list and item names
// from the URL and then delegates to more specific handlers depending on
// the request method.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")
	_, ok := HandledContentTypes[contentType]
	if contentType == "" || !ok {
		contentType = "text/plain"
	}
	r = r.WithContext(context.WithValue(r.Context(), FinalContentTypeKey, contentType))

	w.Header().Set("X-Content-Type-Options", "nosniff")
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
		h.PutHandler(w, r, list, item)
	case "GET":
		h.GetHandler(w, r, list, item)
	case "INCREMENT":
		h.IncHandler(w, r, list, item)
	case "DELETE":
		h.DelHandler(w, r, list, item)
	case "BULKPUT":
		h.BulkPutHandler(w, r, list)
	case "BULKGET":
		h.BulkGetHandler(w, r, list)
	case "BULKINCREMENT":
		h.BulkIncHandler(w, r, list)
	case "BULKDELETE":
		h.BulkDelHandler(w, r, list)
	default:
		http.Error(w, "Unknown method.", http.StatusBadRequest)
	}
}

// PutHandler adds an item to a list. If the list does not already exist,
// the list will be created.
func (h *Handler) PutHandler(w http.ResponseWriter, r *http.Request, list string, item string) {
	count, err := h.Store.Add(r.Context(), list, item)
	if err != nil {
		errStr := fmt.Sprintf("Error trying to add list item: %v", err)
		printError(w, r, &ErrorMessage{Error: errStr}, http.StatusInternalServerError)
		return
	}
	printSuccess(w, r, &AddedMessage{Added: count})
}

// IncHandler increments an item in a list. The returned body text reports
// the number of items found and incremented (1 or 0).
func (h *Handler) IncHandler(w http.ResponseWriter, r *http.Request, list string, item string) {
	count, err := h.Store.Inc(r.Context(), list, item)
	if err != nil {
		errStr := fmt.Sprintf("Error trying to increment list item: %v", err)
		printError(w, r, &ErrorMessage{Error: errStr}, http.StatusInternalServerError)
		return
	}
	printSuccess(w, r, &IncrementedMessage{Incremented: count})
}

// DelHandler deletes an item from a list. The returned body text reports
// the number of items found and deleted (1 or 0).
func (h *Handler) DelHandler(w http.ResponseWriter, r *http.Request, list string, item string) {
	count, err := h.Store.Del(r.Context(), list, item)
	if err != nil {
		errStr := fmt.Sprintf("Error trying to delete list item: %v", err)
		printError(w, r, &ErrorMessage{Error: errStr}, http.StatusInternalServerError)
		return
	}
	printSuccess(w, r, &DeletedMessage{Deleted: count})
}

// GetHandler returns the number of attempts that were made to complete
// an item in a list. When a list or list item is missing, no body will
// be returned, and a status of 404 will be given.
func (h *Handler) GetHandler(w http.ResponseWriter, r *http.Request, list string, item string) {
	attempts, ok, err := h.Store.Get(r.Context(), list, item)
	if err != nil {
		errStr := fmt.Sprintf("Error trying to get list item: %v", err)
		printError(w, r, &ErrorMessage{Error: errStr}, http.StatusInternalServerError)
		return
	}
	if !ok {
		printError(w, r, &ErrorMessage{Error: "Not found."}, http.StatusNotFound)
		return
	}
	// NOTE: taking advantage of the fact that a bare number is valid
	// text/plain as well as valid JSON!
	fmt.Fprintf(w, "%d\n", attempts)
}

func getScrubbedLines(bodyBytes []byte) []string {
	bodyString := string(bodyBytes[:])
	// be nice and trim leading and trailing space from body first.
	bodyString = strings.TrimSpace(bodyString)
	return strings.Split(bodyString, "\n")
}

// BulkPutHandler adds all of the items in the request body (item names
// separated by newlines) to the specified list, and sets their completion
// attempt counts to 0. The response contains the number of items successfully
// inserted, generally len(items) or 0.
func (h *Handler) BulkPutHandler(w http.ResponseWriter, r *http.Request, list string) {
	if r.Body == nil {
		fmt.Fprintf(w, "ADDED 0\n")
		return
	}
	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		errStr := fmt.Sprintf("Error reading body: %v", err)
		printError(w, r, &ErrorMessage{Error: errStr}, http.StatusBadRequest)
		return
	}
	// XXX: Oh, wait; we need a way to parse this regardless if it is
	// text/plain or application/json
	items := getScrubbedLines(bodyBytes)

	count, err := h.Store.BulkAdd(r.Context(), list, items)
	if err != nil {
		errStr := fmt.Sprintf("Error trying to add list items: %v", err)
		printError(w, r, &ErrorMessage{Error: errStr}, http.StatusInternalServerError)
		return
	}
	printSuccess(w, r, &AddedMessage{Added: count})
}

// BulkGetHandler requires the "X-IIDY-Count" header, and takes an optional
// "X-IIDY-After-Item" header. It returns a response body of list items;
// each list item is followed by a space and the number of attempts to
// complete that list item. Each list item / attempt count pair is separated
// by a newline. "X-IIDY-Count" determines how many items are returned (from
// the sorted list). "X-IIDY-After-Item" determines the offset in the list;
// when set to the empty string, we start at the beginning of the list; when
// set to an item (generally the last item from a previous call to this
// handler) we start after that item in the list.
func (h *Handler) BulkGetHandler(w http.ResponseWriter, r *http.Request, list string) {
	startID := r.Header.Get("X-IIDY-After-Item")
	countStr := r.Header.Get("X-IIDY-Count")
	if countStr == "" {
		printError(w, r, &ErrorMessage{Error: "Header not found: X-IIDY-Count"},
			http.StatusBadRequest)
		return
	}
	count, err := strconv.Atoi(countStr)
	if err != nil {
		errStr := fmt.Sprintf("For header X-IIDY-Count, %v is not a number: %v", countStr, err)
		printError(w, r, &ErrorMessage{Error: errStr}, http.StatusInternalServerError)
		return
	}
	if count == 0 {
		return
	}
	listEntries, err := h.Store.BulkGet(r.Context(), list, startID, count)
	if len(listEntries) == 0 {
		// Nothing found, so we are done!
		return
	}
	// Although the client can parse out the last item from the body,
	// as a convenience, also provide the last item in a header.
	w.Header().Set("X-IIDY-Last-Item", listEntries[len(listEntries)-1].Item)
	// XXX START HERE: Need a way to send this as JSON as well as the text/plain
	// we are sending here.
	for _, listItem := range listEntries {
		fmt.Fprintf(w, "%s %d\n", listItem.Item, listItem.Attempts)
	}
}

// BulkIncHandler increments all of the items in the request body (item names
// separated by newlines) in the specified list. The response contains the
// number of items successfully incremented, generally len(items) or 0.
func (h *Handler) BulkIncHandler(w http.ResponseWriter, r *http.Request, list string) {
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
	// XXX: Oh, wait; we need a way to parse this regardless if it is
	// text/plain or application/json
	items := getScrubbedLines(bodyBytes)

	count, err := h.Store.BulkInc(r.Context(), list, items)
	if err != nil {
		errStr := fmt.Sprintf("Error trying to increment list items: %v", err)
		http.Error(w, errStr, http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "INCREMENTED %d\n", count)
}

// BulkDelHandler deletes all of the items in the request body (item names
// separated by newlines) from the specified list. The response contains the
// number of items successfully deleted, generally len(items) or 0.
func (h *Handler) BulkDelHandler(w http.ResponseWriter, r *http.Request, list string) {
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
	// XXX: Oh, wait; we need a way to parse this regardless if it is
	// text/plain or application/json
	items := getScrubbedLines(bodyBytes)

	count, err := h.Store.BulkDel(r.Context(), list, items)
	if err != nil {
		errStr := fmt.Sprintf("Error trying to delete list items: %v", err)
		http.Error(w, errStr, http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "DELETED %d\n", count)
}

func printError(w http.ResponseWriter, r *http.Request, e *ErrorMessage, code int) {
	contentType := r.Context().Value(FinalContentTypeKey)
	if contentType == "application/json" {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(code)
		err := json.NewEncoder(w).Encode(e)
		if err != nil {
			fmt.Printf("Encountered error %v and could not even encode to JSON: %v",
				e, err)
		}
	} else {
		http.Error(w, e.Error, code)
	}
	return
}

func printSuccess(w http.ResponseWriter, r *http.Request, v interface{}) {
	contentType := r.Context().Value(FinalContentTypeKey)
	if contentType == "application/json" {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		err := json.NewEncoder(w).Encode(v)
		if err != nil {
			fmt.Printf("Could not even encode to JSON: %v", v)
		}
	} else {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		switch v.(type) {
		case *AddedMessage:
			m := v.(*AddedMessage)
			fmt.Fprintf(w, "ADDED %d\n", m.Added)
		case *IncrementedMessage:
			m := v.(*IncrementedMessage)
			fmt.Fprintf(w, "INCREMENTED %d\n", m.Incremented)
		case *DeletedMessage:
			m := v.(*DeletedMessage)
			fmt.Fprintf(w, "DELETED %d\n", m.Deleted)
		default:
			fmt.Printf("Could not determine type of: %v", v)
		}
	}
	return
}
