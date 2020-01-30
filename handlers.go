package iidy

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// FinalContentTypeKey is the key to find the ContentType
// in the request's context, after we put it there.
const FinalContentTypeKey string = "final Content-Type"

// BodyBytesKey is the key to find the bytes from the request body
// in the request's context, after we put them there.
const BodyBytesKey string = "bodyBytes"

// QueryKey is the key to find the query parameters
// in the request's context, after we put them there.
const QueryKey string = "query"

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

// ItemListMessage is a list of items that we serialize/deserialize
// to/from JSON when using application/json
type ItemListMessage struct {
	Items []string `json:"items"`
}

// ListEntryMessage is a list of entries and their attempts that we
// serialize/deserialize to/from JSON when using application/json
type ListEntryMessage struct {
	ListEntries []ListEntry `json:"listentries"`
}

// Handler handles requests to "/lists/". It contains an instance of PgStore,
// so that it has a place to store list data.
type Handler struct {
	Store *PgStore
}

// contentTypeHeaderToContext puts the Content-Type header into
// the request's context for use in later steps of processing the
// request. Reading the request can be a stateful matter, so reading
// this header and saving it for later circumvents the "have I already
// read this header?" conundrum.
func contentTypeHeaderToContext(r *http.Request) *http.Request {
	contentType := r.Header.Get("Content-Type")
	_, ok := HandledContentTypes[contentType]
	if contentType == "" || !ok {
		// If the client handed us a content type we do not understand,
		// default to sending and receiving text/plain.
		contentType = "text/plain"
	}
	return r.WithContext(context.WithValue(r.Context(), FinalContentTypeKey, contentType))
}

// requestBodyToContext puts the bytes of the request body into
// the request's context for use in later steps of processing the
// request. Reading the request can be a stateful matter, so reading
// the request body and saving it for later circumvents the "have I already
// read the request body?" conundrum.
func requestBodyToContext(r *http.Request) (*http.Request, error) {
	// Fetch the body now, defensively. Things like r.FormValue
	// can fetch the body, and then subsequent calls to get the body fail.
	if r.Body != nil {
		bodyBytes, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}
		return r.WithContext(context.WithValue(r.Context(), BodyBytesKey, bodyBytes)), nil
	}
	return r, nil
}

// queryParamsToContext parses the query params and makes them available
// in the request's context. Our API only supports query params in the URL,
// not in the request body; the request body is for API payloads that are
// either in text/plain or application/json. In other words, we never parse
// HTTP form vars from the request body.
func queryParamsToContext(r *http.Request) *http.Request {
	query := r.URL.Query()
	return r.WithContext(context.WithValue(r.Context(), QueryKey, query))
}

// ServeHTTP satisfies the http.Handler interface. It is expected to handle
// all traffic to the iidy server. It looks at the request and then delegates to more
// specific handlers depending on the request method.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	r = contentTypeHeaderToContext(r)

	r, err := requestBodyToContext(r)
	if err != nil {
		errStr := fmt.Sprintf("Error reading body: %v", err)
		printError(w, r, &ErrorMessage{Error: errStr}, http.StatusBadRequest)
		return
	}

	r = queryParamsToContext(r)

	// Tell the client to take the "Content-Type header seriously.
	w.Header().Set("X-Content-Type-Options", "nosniff")

	switch r.Method {
	case http.MethodPost:
		h.post(w, r)
	case http.MethodGet:
		h.get(w, r)
	case http.MethodDelete:
		h.delete(w, r)
	default:
		printError(w, r, &ErrorMessage{Error: "Unknown method."}, http.StatusBadRequest)
	}
}

// delete handles DELETEs to these two endpoints:
//     DELETE /v1/lists/<listname>/<itemname>
//     DELETE /v1/bulk/lists/<listname> [itemnames in body]
func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	urlParts := strings.Split(r.URL.Path, "/")
	if len(urlParts) < 6 {
		errStr := fmt.Sprintf(`"%s" is not a valid %s url`, r.URL.Path, http.MethodDelete)
		printError(w, r, &ErrorMessage{Error: errStr}, http.StatusBadRequest)
		return
	}
	if urlParts[3] == "lists" {
		list := urlParts[4]
		item := urlParts[5]
		h.deleteOne(w, r, list, item)
		return
	}
	if urlParts[3] == "bulk" && urlParts[4] == "lists" {
		list := urlParts[5]
		h.deleteMany(w, r, list)
		return
	}
	errStr := fmt.Sprintf(`"%s" is not a valid %s url`, r.URL.Path, http.MethodDelete)
	printError(w, r, &ErrorMessage{Error: errStr}, http.StatusBadRequest)
	return
}

// get handles GETs to these two endpoints:
//     GET /iidy/v1/lists/<listname>/<itemname>
//     GET /iidy/v1/bulk/lists/<listname>?count=ct&after_id=it
func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	urlParts := strings.Split(r.URL.Path, "/")
	if len(urlParts) < 6 {
		errStr := fmt.Sprintf(`"%s" is not a valid %s url`, r.URL.Path, http.MethodGet)
		printError(w, r, &ErrorMessage{Error: errStr}, http.StatusBadRequest)
		return
	}
	if urlParts[3] == "lists" {
		list := urlParts[4]
		item := urlParts[5]
		h.getOne(w, r, list, item)
		return
	}
	if urlParts[3] == "bulk" && urlParts[4] == "lists" {
		list := urlParts[5]
		h.getMany(w, r, list)
		return
	}
	errStr := fmt.Sprintf(`"%s" is not a valid %s url`, r.URL.Path, http.MethodPost)
	printError(w, r, &ErrorMessage{Error: errStr}, http.StatusBadRequest)
	return
}

// post handles POSTs to these three endpoints:
//     POST /iidy/v1/lists/<listname>/<itemname>
//     POST /iidy/v1/bulk/lists/<listname> [itemnames in body]
//     POST /iidy/v1/bulk/lists/<listname>?action=increment [itemnames in body]
func (h *Handler) post(w http.ResponseWriter, r *http.Request) {
	urlParts := strings.Split(r.URL.Path, "/")
	if len(urlParts) < 6 {
		errStr := fmt.Sprintf(`"%s" is not a valid %s url`, r.URL.Path, http.MethodPost)
		printError(w, r, &ErrorMessage{Error: errStr}, http.StatusBadRequest)
		return
	}

	query := r.Context().Value(QueryKey).(url.Values)

	if urlParts[3] == "lists" {
		list := urlParts[4]
		item := urlParts[5]
		if query.Get("action") == "increment" {
			h.incrementOne(w, r, list, item)
		} else {
			h.insertOne(w, r, list, item)
		}
		return
	}
	if urlParts[3] == "bulk" && urlParts[4] == "lists" {
		list := urlParts[5]
		if query.Get("action") == "increment" {
			h.incrementMany(w, r, list)
		} else {
			h.insertMany(w, r, list)
		}
		return
	}
	errStr := fmt.Sprintf(`"%s" is not a valid %s url`, r.URL.Path, http.MethodPost)
	printError(w, r, &ErrorMessage{Error: errStr}, http.StatusBadRequest)
	return
}

// insertOne adds an item to a list. If the list does not already exist,
// the list will be created.
func (h *Handler) insertOne(w http.ResponseWriter, r *http.Request, list string, item string) {
	count, err := h.Store.InsertOne(r.Context(), list, item)
	if err != nil {
		errStr := fmt.Sprintf("Error trying to add list item: %v", err)
		printError(w, r, &ErrorMessage{Error: errStr}, http.StatusInternalServerError)
		return
	}
	printSuccess(w, r, &AddedMessage{Added: count}, http.StatusCreated)
}

// incrementOne increments an item in a list. The returned body text reports
// the number of items found and incremented (1 or 0).
func (h *Handler) incrementOne(w http.ResponseWriter, r *http.Request, list string, item string) {
	count, err := h.Store.Inc(r.Context(), list, item)
	if err != nil {
		errStr := fmt.Sprintf("Error trying to increment list item: %v", err)
		printError(w, r, &ErrorMessage{Error: errStr}, http.StatusInternalServerError)
		return
	}
	printSuccess(w, r, &IncrementedMessage{Incremented: count}, http.StatusOK)
}

// deleteOne deletes an item from a list. The returned body text reports
// the number of items found and deleted (1 or 0).
func (h *Handler) deleteOne(w http.ResponseWriter, r *http.Request, list string, item string) {
	count, err := h.Store.Del(r.Context(), list, item)
	if err != nil {
		errStr := fmt.Sprintf("Error trying to delete list item: %v", err)
		printError(w, r, &ErrorMessage{Error: errStr}, http.StatusInternalServerError)
		return
	}
	printSuccess(w, r, &DeletedMessage{Deleted: count}, http.StatusOK)
}

// getOne returns the number of attempts that were made to complete
// an item in a list. When a list or list item is missing, no body will
// be returned, and a status of 404 will be given.
func (h *Handler) getOne(w http.ResponseWriter, r *http.Request, list string, item string) {
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
	printSuccess(w, r, &ListEntry{Item: item, Attempts: attempts}, http.StatusOK)
}

// getItemsFromBody gets a slice of list items from the request body,
// regardless if the request body is in JSON or plain text format.
func getItemsFromBody(contentType string, bodyBytes []byte) ([]string, error) {
	if bodyBytes == nil || len(bodyBytes) == 0 {
		return nil, nil
	}
	if contentType == "application/json" {
		return getItemsFromJSON(bodyBytes)
	}
	// default to text/plain
	return getItemsFromPlainText(bodyBytes), nil
}

// getItemsFromJSON gets a slice of list item names from
// the bytes of a request body that is in JSON format.
func getItemsFromJSON(bodyBytes []byte) ([]string, error) {
	if bodyBytes == nil || len(bodyBytes) == 0 {
		return nil, nil
	}
	var msg *ItemListMessage
	err := json.Unmarshal(bodyBytes, &msg)
	if err != nil {
		return nil, err
	}
	return msg.Items, nil
}

// getItemsFromPlainText gets a slice of list item names from
// the bytes of a request body that is in plain text format.
func getItemsFromPlainText(bodyBytes []byte) []string {
	if bodyBytes == nil || len(bodyBytes) == 0 {
		return nil
	}
	bodyString := string(bodyBytes[:])
	// be nice and trim leading and trailing space from body first.
	bodyString = strings.TrimSpace(bodyString)
	return strings.Split(bodyString, "\n")
}

// insertMany adds all of the items in the request body to the specified
// list, and sets their completion attempt counts to 0. The response contains
// the number of items successfully inserted, generally len(items) or 0.
func (h *Handler) insertMany(w http.ResponseWriter, r *http.Request, list string) {
	v := r.Context().Value(BodyBytesKey)
	if v == nil {
		printSuccess(w, r, &AddedMessage{Added: 0}, http.StatusOK)
		return
	}
	bodyBytes := v.([]byte)
	items, err := getItemsFromBody(fmt.Sprintf("%s", r.Context().Value(FinalContentTypeKey)), bodyBytes)
	if err != nil {
		errStr := fmt.Sprintf("Error trying to parse list of items from request body: %v", err)
		printError(w, r, &ErrorMessage{Error: errStr}, http.StatusInternalServerError)
		return
	}

	count, err := h.Store.InsertMany(r.Context(), list, items)
	if err != nil {
		errStr := fmt.Sprintf("Error trying to add list items: %v", err)
		printError(w, r, &ErrorMessage{Error: errStr}, http.StatusInternalServerError)
		return
	}
	printSuccess(w, r, &AddedMessage{Added: count}, http.StatusCreated)
}

// getMany requires the "count" query arg, and takes an optional
// "after_id" query arg. It returns a response body of list items;
// each list item shows the number of attempts to
// complete that list item. "count" determines how many items are
// returned (from
// the sorted list). "after_id" determines the offset in the list;
// when set to the empty string, we start at the beginning of the list; when
// set to an item (generally the last item from a previous call to this
// handler) we start after that item in the list.
func (h *Handler) getMany(w http.ResponseWriter, r *http.Request, list string) {
	query := r.Context().Value(QueryKey).(url.Values)
	afterID := query.Get("after_id")
	countStr := query.Get("count")
	if countStr == "" {
		printError(w, r, &ErrorMessage{Error: "Query arg not found: count"},
			http.StatusBadRequest)
		return
	}
	count, err := strconv.Atoi(countStr)
	if err != nil {
		errStr := fmt.Sprintf("For query arg count, %v is not a number: %v", countStr, err)
		printError(w, r, &ErrorMessage{Error: errStr}, http.StatusInternalServerError)
		return
	}
	if count == 0 {
		return
	}
	listEntries, err := h.Store.BulkGet(r.Context(), list, afterID, count)
	if len(listEntries) == 0 {
		// Nothing found, so we are done!
		return
	}
	// Although the client can parse out the last item from the body,
	// as a convenience, also provide the last item in a header.
	w.Header().Set("X-IIDY-Last-Item", listEntries[len(listEntries)-1].Item)
	printListEntries(w, r, listEntries)
}

// incrementMany increments all of the items in the request body
// in the specified list. The response contains the
// number of items successfully incremented, generally len(items) or 0.
func (h *Handler) incrementMany(w http.ResponseWriter, r *http.Request, list string) {
	v := r.Context().Value(BodyBytesKey)
	if v == nil {
		printSuccess(w, r, &IncrementedMessage{Incremented: 0}, http.StatusOK)
		return
	}
	bodyBytes := v.([]byte)
	items, err := getItemsFromBody(fmt.Sprintf("%s", r.Context().Value(FinalContentTypeKey)), bodyBytes)
	if err != nil {
		errStr := fmt.Sprintf("Error trying to parse list of items from request body: %v", err)
		printError(w, r, &ErrorMessage{Error: errStr}, http.StatusInternalServerError)
		return
	}

	count, err := h.Store.BulkInc(r.Context(), list, items)
	if err != nil {
		errStr := fmt.Sprintf("Error trying to increment list items: %v", err)
		http.Error(w, errStr, http.StatusInternalServerError)
		return
	}
	printSuccess(w, r, &IncrementedMessage{Incremented: count}, http.StatusOK)
}

// deleteMany deletes all of the items in the request body
// from the specified list. The response contains the
// number of items successfully deleted, generally len(items) or 0.
func (h *Handler) deleteMany(w http.ResponseWriter, r *http.Request, list string) {
	v := r.Context().Value(BodyBytesKey)
	if v == nil {
		printSuccess(w, r, &DeletedMessage{Deleted: 0}, http.StatusOK)
		return
	}
	bodyBytes := v.([]byte)
	items, err := getItemsFromBody(fmt.Sprintf("%s", r.Context().Value(FinalContentTypeKey)), bodyBytes)
	if err != nil {
		errStr := fmt.Sprintf("Error trying to parse list of items from request body: %v", err)
		printError(w, r, &ErrorMessage{Error: errStr}, http.StatusInternalServerError)
		return
	}

	count, err := h.Store.BulkDel(r.Context(), list, items)
	if err != nil {
		errStr := fmt.Sprintf("Error trying to delete list items: %v", err)
		http.Error(w, errStr, http.StatusInternalServerError)
		return
	}
	printSuccess(w, r, &DeletedMessage{Deleted: count}, http.StatusOK)
}

// printListEntries prints list entries to the w, the response writer.
// This function correctly determines whether JSON or plain text is
// requested.
func printListEntries(w http.ResponseWriter, r *http.Request, listEntries []ListEntry) {
	contentType := r.Context().Value(FinalContentTypeKey)
	if contentType == "application/json" {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		err := json.NewEncoder(w).Encode(&ListEntryMessage{ListEntries: listEntries})
		if err != nil {
			fmt.Printf("Could not encode list entries to JSON: %v", err)
		}
	} else {
		for _, listItem := range listEntries {
			fmt.Fprintf(w, "%s %d\n", listItem.Item, listItem.Attempts)
		}
	}
	return
}

// printError prints an error to w, the response writer, in the requested
// format, JSON or plain text. The response code is also set as specified.
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

// printSuccess prints a success message to w, the response writer, in the requested
// format, JSON or plain text. The response code is also set as specified.
func printSuccess(w http.ResponseWriter, r *http.Request, v interface{}, code int) {
	w.WriteHeader(code)
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
		case *ListEntry:
			m := v.(*ListEntry)
			fmt.Fprintf(w, "%d\n", m.Attempts)
		default:
			fmt.Printf("Could not determine type of: %v", v)
		}
	}
	return
}
