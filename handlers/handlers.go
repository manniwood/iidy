package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/manniwood/iidy/data"
)

// HandledContentTypes are the content types handled
// by this service.
var HandledContentTypes = map[string]struct{}{
	"text/plain":       {},
	"application/json": {},
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
	ListEntries []data.ListEntry `json:"listentries"`
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

// InsertOne adds an item to a list. If the list does not already exist,
// the list will be created.
func InsertOne(w http.ResponseWriter, r *http.Request) {
	list := r.PathValue("list")
	item := r.PathValue("item")

	contentType := r.Header.Get("Content-Type")
	_, ok := HandledContentTypes[contentType]
	if contentType == "" || !ok {
		// If the client handed us a content type we do not understand,
		// default to sending and receiving text/plain.
		contentType = "text/plain"
	}

	// Tell the client to take the "Content-Type header seriously.
	w.Header().Set("X-Content-Type-Options", "nosniff")

	count, err := data.InsertOne(r.Context(), data.PgxPool, list, item)
	if err != nil {
		errStr := fmt.Sprintf("Error trying to add list item: %v", err)
		printError(w, contentType, &ErrorMessage{Error: errStr}, http.StatusInternalServerError)
		return
	}
	printSuccess(w, contentType, &AddedMessage{Added: count}, http.StatusCreated)
}

// GetOne returns the number of attempts that were made to complete
// an item in a list. When a list or list item is missing, no body will
// be returned, and a status of 404 will be given.
func GetOne(w http.ResponseWriter, r *http.Request) {
	list := r.PathValue("list")
	item := r.PathValue("item")

	contentType := r.Header.Get("Content-Type")
	_, ok := HandledContentTypes[contentType]
	if contentType == "" || !ok {
		// If the client handed us a content type we do not understand,
		// default to sending and receiving text/plain.
		contentType = "text/plain"
	}

	// Tell the client to take the "Content-Type header seriously.
	w.Header().Set("X-Content-Type-Options", "nosniff")

	attempts, ok, err := data.GetOne(r.Context(), data.PgxPool, list, item)
	if err != nil {
		errStr := fmt.Sprintf("Error trying to get list item: %v", err)
		printError(w, contentType, &ErrorMessage{Error: errStr}, http.StatusInternalServerError)
		return
	}
	if !ok {
		printError(w, contentType, &ErrorMessage{Error: "Not found."}, http.StatusNotFound)
		return
	}
	printSuccess(w, contentType, &data.ListEntry{Item: item, Attempts: attempts}, http.StatusOK)
}

// DeleteOne deletes an item from a list. The returned body text reports
// the number of items found and deleted (1 or 0).
func DeleteOne(w http.ResponseWriter, r *http.Request) {
	list := r.PathValue("list")
	item := r.PathValue("item")

	contentType := r.Header.Get("Content-Type")
	_, ok := HandledContentTypes[contentType]
	if contentType == "" || !ok {
		// If the client handed us a content type we do not understand,
		// default to sending and receiving text/plain.
		contentType = "text/plain"
	}

	// Tell the client to take the "Content-Type header seriously.
	w.Header().Set("X-Content-Type-Options", "nosniff")

	count, err := data.DeleteOne(r.Context(), data.PgxPool, list, item)
	if err != nil {
		errStr := fmt.Sprintf("Error trying to delete list item: %v", err)
		printError(w, contentType, &ErrorMessage{Error: errStr}, http.StatusInternalServerError)
		return
	}
	printSuccess(w, contentType, &DeletedMessage{Deleted: count}, http.StatusOK)
}

// IncrementOne increments an item in a list. The returned body text reports
// the number of items found and incremented (1 or 0).
func IncrementOne(w http.ResponseWriter, r *http.Request) {
	list := r.PathValue("list")
	item := r.PathValue("item")

	contentType := r.Header.Get("Content-Type")
	_, ok := HandledContentTypes[contentType]
	if contentType == "" || !ok {
		// If the client handed us a content type we do not understand,
		// default to sending and receiving text/plain.
		contentType = "text/plain"
	}

	// Tell the client to take the "Content-Type header seriously.
	w.Header().Set("X-Content-Type-Options", "nosniff")

	count, err := data.IncrementOne(r.Context(), data.PgxPool, list, item)
	if err != nil {
		errStr := fmt.Sprintf("Error trying to increment list item: %v", err)
		printError(w, contentType, &ErrorMessage{Error: errStr}, http.StatusInternalServerError)
		return
	}
	printSuccess(w, contentType, &IncrementedMessage{Incremented: count}, http.StatusOK)
}

// InsertBatch adds all of the items in the request body to the specified
// list, and sets their completion attempt counts to 0. The response contains
// the number of items successfully inserted, generally len(items) or 0.
func InsertBatch(w http.ResponseWriter, r *http.Request) {
	list := r.PathValue("list")

	contentType := r.Header.Get("Content-Type")
	_, ok := HandledContentTypes[contentType]
	if contentType == "" || !ok {
		// If the client handed us a content type we do not understand,
		// default to sending and receiving text/plain.
		contentType = "text/plain"
	}

	var bodyBytes []byte
	var err error
	if r.Body != nil {
		bodyBytes, err = io.ReadAll(r.Body)
		if err != nil {
			errStr := fmt.Sprintf("Error trying to read request body: %v", err)
			printError(w, contentType, &ErrorMessage{Error: errStr}, http.StatusInternalServerError)
			return
		}
	}

	// Tell the client to take the "Content-Type header seriously.
	w.Header().Set("X-Content-Type-Options", "nosniff")

	if bodyBytes == nil {
		printSuccess(w, contentType, &AddedMessage{Added: 0}, http.StatusOK)
		return
	}
	items, err := getItemsFromBody(contentType, bodyBytes)
	if err != nil {
		errStr := fmt.Sprintf("Error trying to parse list of items from request body: %v", err)
		printError(w, contentType, &ErrorMessage{Error: errStr}, http.StatusInternalServerError)
		return
	}

	count, err := data.InsertBatch(r.Context(), data.PgxPool, list, items)
	if err != nil {
		errStr := fmt.Sprintf("Error trying to add list items: %v", err)
		printError(w, contentType, &ErrorMessage{Error: errStr}, http.StatusInternalServerError)
		return
	}
	printSuccess(w, contentType, &AddedMessage{Added: count}, http.StatusCreated)
}

// DeleteBatch deletes all of the items in the request body
// from the specified list. The response contains the
// number of items successfully deleted, generally len(items) or 0.
func DeleteBatch(w http.ResponseWriter, r *http.Request) {
	list := r.PathValue("list")

	contentType := r.Header.Get("Content-Type")
	_, ok := HandledContentTypes[contentType]
	if contentType == "" || !ok {
		// If the client handed us a content type we do not understand,
		// default to sending and receiving text/plain.
		contentType = "text/plain"
	}

	var bodyBytes []byte
	var err error
	if r.Body != nil {
		bodyBytes, err = io.ReadAll(r.Body)
		if err != nil {
			errStr := fmt.Sprintf("Error trying to read request body: %v", err)
			printError(w, contentType, &ErrorMessage{Error: errStr}, http.StatusInternalServerError)
			return
		}
	}

	// Tell the client to take the "Content-Type header seriously.
	w.Header().Set("X-Content-Type-Options", "nosniff")

	if bodyBytes == nil {
		printSuccess(w, contentType, &AddedMessage{Added: 0}, http.StatusOK)
		return
	}

	items, err := getItemsFromBody(contentType, bodyBytes)
	if err != nil {
		errStr := fmt.Sprintf("Error trying to parse list of items from request body: %v", err)
		printError(w, contentType, &ErrorMessage{Error: errStr}, http.StatusInternalServerError)
		return
	}

	count, err := data.DeleteBatch(r.Context(), data.PgxPool, list, items)
	if err != nil {
		errStr := fmt.Sprintf("Error trying to delete list items: %v", err)
		printError(w, contentType, &ErrorMessage{Error: errStr}, http.StatusInternalServerError)
		return
	}
	printSuccess(w, contentType, &DeletedMessage{Deleted: count}, http.StatusOK)
}

// GetBatch requires the "count" query arg, and takes an optional
// "after_id" query arg. It returns a response body of list items;
// each list item shows the number of attempts to
// complete that list item. "count" determines how many items are
// returned (from
// the sorted list). "after_id" determines the offset in the list;
// when set to the empty string, we start at the beginning of the list; when
// set to an item (generally the last item from a previous call to this
// handler) we start after that item in the list.
func GetBatch(w http.ResponseWriter, r *http.Request) {
	list := r.PathValue("list")

	contentType := r.Header.Get("Content-Type")
	_, ok := HandledContentTypes[contentType]
	if contentType == "" || !ok {
		// If the client handed us a content type we do not understand,
		// default to sending and receiving text/plain.
		contentType = "text/plain"
	}

	query := r.URL.Query()
	afterID := query.Get("after_id")
	countStr := query.Get("count")

	// Tell the client to take the "Content-Type header seriously.
	w.Header().Set("X-Content-Type-Options", "nosniff")

	if countStr == "" {
		printError(w, contentType, &ErrorMessage{Error: "Query arg not found: count"},
			http.StatusBadRequest)
		return
	}
	count, err := strconv.Atoi(countStr)
	if err != nil {
		errStr := fmt.Sprintf("For query arg count, %v is not a number: %v", countStr, err)
		printError(w, contentType, &ErrorMessage{Error: errStr}, http.StatusInternalServerError)
		return
	}
	if count == 0 {
		w.WriteHeader(http.StatusOK)
		return
	}
	listEntries, err := data.GetBatch(r.Context(), data.PgxPool, list, afterID, count)
	if len(listEntries) == 0 {
		// Nothing found, so we are done!
		w.WriteHeader(http.StatusOK)
		return
	}
	// Although the client can parse out the last item from the body,
	// as a convenience, also provide the last item in a header.
	w.Header().Set("X-IIDY-Last-Item", listEntries[len(listEntries)-1].Item)
	printListEntries(w, contentType, listEntries)
}

// IncrementBatch increments all of the items in the request body
// in the specified list. The response contains the
// number of items successfully incremented, generally len(items) or 0.
func IncrementBatch(w http.ResponseWriter, r *http.Request) {
	list := r.PathValue("list")

	contentType := r.Header.Get("Content-Type")
	_, ok := HandledContentTypes[contentType]
	if contentType == "" || !ok {
		// If the client handed us a content type we do not understand,
		// default to sending and receiving text/plain.
		contentType = "text/plain"
	}

	var bodyBytes []byte
	var err error
	if r.Body != nil {
		bodyBytes, err = io.ReadAll(r.Body)
		if err != nil {
			errStr := fmt.Sprintf("Error trying to read request body: %v", err)
			printError(w, contentType, &ErrorMessage{Error: errStr}, http.StatusInternalServerError)
			return
		}
	}

	// Tell the client to take the "Content-Type header seriously.
	w.Header().Set("X-Content-Type-Options", "nosniff")

	if bodyBytes == nil {
		printSuccess(w, contentType, &AddedMessage{Added: 0}, http.StatusOK)
		return
	}
	items, err := getItemsFromBody(contentType, bodyBytes)
	if err != nil {
		errStr := fmt.Sprintf("Error trying to parse list of items from request body: %v", err)
		printError(w, contentType, &ErrorMessage{Error: errStr}, http.StatusInternalServerError)
		return
	}

	count, err := data.IncrementBatch(r.Context(), data.PgxPool, list, items)
	if err != nil {
		errStr := fmt.Sprintf("Error trying to increment list items: %v", err)
		printError(w, contentType, &ErrorMessage{Error: errStr}, http.StatusInternalServerError)
		return
	}
	printSuccess(w, contentType, &IncrementedMessage{Incremented: count}, http.StatusOK)
}

// printError prints an error to w, the response writer, in the requested
// format, JSON or plain text. The response code is also set as specified.
func printError(w http.ResponseWriter, contentType string, e *ErrorMessage, code int) {
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
func printSuccess(w http.ResponseWriter, contentType string, v interface{}, code int) {
	w.WriteHeader(code)
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
		case *data.ListEntry:
			m := v.(*data.ListEntry)
			fmt.Fprintf(w, "%d\n", m.Attempts)
		default:
			fmt.Printf("Could not determine type of: %v", v)
		}
	}
	return
}

// printListEntries prints list entries to the w, the response writer.
// This function correctly determines whether JSON or plain text is
// requested.
func printListEntries(w http.ResponseWriter, contentType string, listEntries []data.ListEntry) {
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
