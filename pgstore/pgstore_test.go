package pgstore

import (
	"context"
	"reflect"
	"testing"
)

func GetEmptyStore(t *testing.T) *PgStore {
	p, err := NewPgStore("")
	if err != nil {
		t.Errorf("Error instantiating PgStore: %v", err)
	}
	p.Nuke(context.Background())
	return p
}

// Our tests add this test item over and over,
// so here it is.
func addSingleStartingItem(t *testing.T, s *PgStore) {
	count, err := s.InsertOne(context.Background(), "downloads", "kernel.tar.gz")
	if err != nil {
		t.Errorf("Error adding item: %v", err)
	}
	if count != 1 {
		t.Error("Did not properly add item to list.")
	}
}

func TestInsertOneAndGetOne(t *testing.T) {
	s := GetEmptyStore(t)
	addSingleStartingItem(t, s)

	// Did we really add the item?
	attempts, ok, err := s.GetOne(context.Background(), "downloads", "kernel.tar.gz")
	if err != nil {
		t.Errorf("Error getting item: %v", err)
	}
	if attempts != 0 {
		t.Error("attempts != 0")
	}
	if !ok {
		t.Error("Did not properly add item to list.")
	}
}

func TestUnhappyGetOneScenarios(t *testing.T) {
	s := GetEmptyStore(t)

	// What about getting an item that doesn't exist?
	_, ok, err := s.GetOne(context.Background(), "downloads", "I do not exist")
	if err != nil {
		t.Errorf("Error getting item: %v", err)
	}
	if ok {
		t.Error("List claims to return value that was not added to list.")
	}

	// What about getting from a list that doesn't exixt?
	_, ok, err = s.GetOne(context.Background(), "I do not exist", "kernel.tar.gz")
	if err != nil {
		t.Errorf("Error getting item: %v", err)
	}
	if ok {
		t.Error("Non-existent list claims to return value.")
	}
}

func TestDeleteOne(t *testing.T) {
	s := GetEmptyStore(t)
	addSingleStartingItem(t, s)

	// Can we successfully delete?
	count, err := s.DeleteOne(context.Background(), "downloads", "kernel.tar.gz")
	if err != nil {
		t.Errorf("Error trying to delete item from list: %v", err)
	}
	if count != 1 {
		t.Error("Did not properly delete item from list.")
	}

	// Does getting the deleted value correctly return nothing?
	_, ok, err := s.GetOne(context.Background(), "downloads", "kernel.tar.gz")
	if err != nil {
		t.Errorf("Error getting item: %v", err)
	}
	if ok {
		t.Error("Did not properly delete item to list.")
	}

	// What about deleting an item that isn't there?
	count, err = s.DeleteOne(context.Background(), "downloads", "I do not exist")
	if err != nil {
		t.Errorf("Error trying to delete item from list: %v", err)
	}
	if count != 0 {
		t.Error("Did not properly report non-deletion of item.")
	}

	// What about deleting an item from a list that isn't there?
	count, err = s.DeleteOne(context.Background(), "I do not exist", "kernel.tar.gz")
	if err != nil {
		t.Errorf("Error trying to delete item from non-existent list: %v", err)
	}
	if count != 0 {
		t.Error("Did not properly report non-deletion of item from no-existent list.")
	}
}

func TestIncrementOne(t *testing.T) {
	s := GetEmptyStore(t)
	addSingleStartingItem(t, s)

	// Does incrementing an item's attempts work?
	count, err := s.IncrementOne(context.Background(), "downloads", "kernel.tar.gz")
	if err != nil {
		t.Errorf("Error trying to increment: %v", err)
	}
	if count != 1 {
		t.Error("Did not properly increment.")
	}

	// When we get the incremented attempt, is is correct?
	attempts, ok, err := s.GetOne(context.Background(), "downloads", "kernel.tar.gz")
	if err != nil {
		t.Errorf("Error getting item: %v", err)
	}
	if !ok {
		t.Error("Did not properly add item to list.")
	}
	if attempts != 1 {
		t.Error("Did not properly increment item in list.")
	}

	// What about incrementing an item that's not there?
	count, err = s.IncrementOne(context.Background(), "downloads", "I do not exist")
	if err != nil {
		t.Errorf("Error trying to increment item from list: %v", err)
	}
	if count != 0 {
		t.Error("Did not properly report non-increment of item.")
	}

	// What about incrementing an item from a list that's not there?
	count, err = s.IncrementOne(context.Background(), "I do not exist", "kernel.tar.gz")
	if err != nil {
		t.Errorf("Error trying to increment item from list: %v", err)
	}
	if count != 0 {
		t.Error("Did not properly report non-increment of item from non-existent list.")
	}
}

func TestInsertBatch(t *testing.T) {
	s := GetEmptyStore(t)
	files := []string{"kernel.tar.gz", "vim.tar.gz", "robots.txt"}

	// Does batch add work?
	count, err := s.InsertBatch(context.Background(), "downloads", files)
	if err != nil {
		t.Errorf("Error batch inserting: %v", err)
	}
	if count != 3 {
		t.Errorf("Batch incremented wrong number of items. Expected 5, got %v", count)
	}

	// If we get the list items, do they exist?
	for _, file := range files {
		attempts, ok, err := s.GetOne(context.Background(), "downloads", file)
		if err != nil {
			t.Errorf("Error getting item: %v", err)
		}
		if attempts != 0 {
			t.Errorf("Attempts for freshly-created %v is not 0", file)
		}
		if !ok {
			t.Error("Did not properly add item to list.")
		}
	}

	// What if we batch put nothing?
	count, err = s.InsertBatch(context.Background(), "downloads", []string{})
	if err != nil {
		t.Errorf("Error batch inserting: %v", err)
	}
	if count != 0 {
		t.Errorf("Batch added wrong number of items. Expected 0, got %v", count)
	}
}

// These items are expected to be in the db at the start
// of the next few batch tests.
func batchAddTestItems(t *testing.T, s *PgStore) {
	// Batch add a bunch of test items.
	files := []string{"a", "b", "c", "d", "e", "f", "g"}
	count, err := s.InsertBatch(context.Background(), "downloads", files)
	if err != nil {
		t.Errorf("Error batch inserting: %v", err)
	}
	if count != 7 {
		t.Errorf("Batch added wrong number of items. Expected 5, got %v", count)
	}
}

func TestGetBatch(t *testing.T) {
	var tests = []struct {
		afterItem string
		want      []ListEntry
	}{
		{"", []ListEntry{{"a", 0}, {"b", 0}}},
		{"b", []ListEntry{{"c", 0}, {"d", 0}}},
		{"d", []ListEntry{{"e", 0}, {"f", 0}}},
		{"f", []ListEntry{{"g", 0}}},
	}
	s := GetEmptyStore(t)
	batchAddTestItems(t, s)

	// If we batch get 2 items at a time, does everything work?
	for _, test := range tests {
		items, err := s.GetBatch(context.Background(), "downloads", test.afterItem, 2)
		if err != nil {
			t.Errorf("Error batch fetching: %v", err)
		}
		if !reflect.DeepEqual(test.want, items) {
			t.Errorf("Expected %v; got %v", test.want, items)
		}
	}

	// What if we batch get nothing?
	items, err := s.GetBatch(context.Background(), "downloads", "", 0)
	if err != nil {
		t.Errorf("Error batch deleting: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("Batch get of nothing yeilded results!")
	}
}

func TestIncrementBatch(t *testing.T) {
	s := GetEmptyStore(t)
	batchAddTestItems(t, s)

	// Does batch increment work?
	count, err := s.IncrementBatch(context.Background(), "downloads", []string{"a", "b", "c", "d", "e"})
	if err != nil {
		t.Errorf("Error batch incrementing: %v", err)
	}
	if count != 5 {
		t.Errorf("Batch incremented wrong number of items. Expected 5, got %v", count)
	}

	// If we look for incremented items, are they incremented?
	for _, file := range []string{"a", "b", "c", "d", "e"} {
		attempts, ok, err := s.GetOne(context.Background(), "downloads", file)
		if err != nil {
			t.Errorf("Error getting item: %v", err)
		}
		if !ok {
			t.Errorf("Did not properly get item %v from list.", file)
		}
		if attempts != 1 {
			t.Errorf("Did not properly increment item %v.", file)
		}
	}

	// What about non-incremented items? Were they left alone?
	for _, file := range []string{"f", "g"} {
		attempts, ok, err := s.GetOne(context.Background(), "downloads", file)
		if err != nil {
			t.Errorf("Error getting item: %v", err)
		}
		if !ok {
			t.Errorf("Did not properly get item %v from list.", file)
		}
		if attempts != 0 {
			t.Errorf("Item %v is incorrectly incremented.", file)
		}
	}

	// What if we batch increment nothing?
	count, err = s.IncrementBatch(context.Background(), "downloads", []string{})
	if err != nil {
		t.Errorf("Error batch deleting: %v", err)
	}
	if count != 0 {
		t.Errorf("Batch incremented wrong number of items. Expected 0, got %v", count)
	}
}

func TestDeleteBatch(t *testing.T) {
	s := GetEmptyStore(t)
	batchAddTestItems(t, s)

	// Does batch delete work?
	count, err := s.DeleteBatch(context.Background(), "downloads", []string{"a", "b", "c", "d", "e"})
	if err != nil {
		t.Errorf("Error batch deleting: %v", err)
	}
	if count != 5 {
		t.Errorf("Batch deleted wrong number of items. Expected 5, got %v", count)
	}

	// If we look for the deleted items, are they correctly missing?
	for _, file := range []string{"a", "b", "c", "d", "e"} {
		_, ok, err := s.GetOne(context.Background(), "downloads", file)
		if err != nil {
			t.Errorf("Error getting item: %v", err)
		}
		if ok {
			t.Errorf("Found item %v that should have been deleted from list.", file)
		}
	}

	// Were other items left alone?
	for _, file := range []string{"f", "g"} {
		attempts, ok, err := s.GetOne(context.Background(), "downloads", file)
		if err != nil {
			t.Errorf("Error getting item: %v", err)
		}
		if !ok {
			t.Errorf("Item %v should not have been deleted from list.", file)
		}
		if attempts != 0 {
			t.Errorf("Item %v is incorrectly incremented.", file)
		}
	}

	// What if we batch delete nothing?
	count, err = s.DeleteBatch(context.Background(), "downloads", []string{})
	if err != nil {
		t.Errorf("Error batch deleting: %v", err)
	}
	if count != 0 {
		t.Errorf("Batch deleted wrong number of items. Expected 0, got %v", count)
	}
}
