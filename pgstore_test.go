package iidy

import (
	"context"
	"reflect"
	"testing"
)

func getEmptyStore(t *testing.T) *PgStore {
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
	count, err := s.Add(context.Background(), "downloads", "kernel.tar.gz")
	if err != nil {
		t.Errorf("Error adding item: %v", err)
	}
	if count != 1 {
		t.Error("Did not properly add item to list.")
	}
}

func TestUnhappyGetScenarios(t *testing.T) {
	s := getEmptyStore(t)

	// What about getting an item that doesn't exist?
	_, ok, err := s.Get(context.Background(), "downloads", "I do not exist")
	if err != nil {
		t.Errorf("Error getting item: %v", err)
	}
	if ok {
		t.Error("List claims to return value that was not added to list.")
	}

	// What about getting from a list that doesn't exixt?
	_, ok, err = s.Get(context.Background(), "I do not exist", "kernel.tar.gz")
	if err != nil {
		t.Errorf("Error getting item: %v", err)
	}
	if ok {
		t.Error("Non-existent list claims to return value.")
	}
}

func TestDel(t *testing.T) {
	s := getEmptyStore(t)
	addSingleStartingItem(t, s)

	// Can we successfully delete?
	count, err := s.Del(context.Background(), "downloads", "kernel.tar.gz")
	if err != nil {
		t.Errorf("Error trying to delete item from list: %v", err)
	}
	if count != 1 {
		t.Error("Did not properly delete item from list.")
	}

	// Does getting the deleted value correctly return nothing?
	_, ok, err := s.Get(context.Background(), "downloads", "kernel.tar.gz")
	if err != nil {
		t.Errorf("Error getting item: %v", err)
	}
	if ok {
		t.Error("Did not properly delete item to list.")
	}

	// What about deleting an item that isn't there?
	count, err = s.Del(context.Background(), "downloads", "I do not exist")
	if err != nil {
		t.Errorf("Error trying to delete item from list: %v", err)
	}
	if count != 0 {
		t.Error("Did not properly report non-deletion of item.")
	}

	// What about deleting an item from a list that isn't there?
	count, err = s.Del(context.Background(), "I do not exist", "kernel.tar.gz")
	if err != nil {
		t.Errorf("Error trying to delete item from non-existent list: %v", err)
	}
	if count != 0 {
		t.Error("Did not properly report non-deletion of item from no-existent list.")
	}
}

func TestInc(t *testing.T) {
	s := getEmptyStore(t)
	addSingleStartingItem(t, s)

	// Does incrementing an item's attempts work?
	count, err := s.Inc(context.Background(), "downloads", "kernel.tar.gz")
	if err != nil {
		t.Errorf("Error trying to increment: %v", err)
	}
	if count != 1 {
		t.Error("Did not properly increment.")
	}

	// When we get the incremented attempt, is is correct?
	attempts, ok, err := s.Get(context.Background(), "downloads", "kernel.tar.gz")
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
	count, err = s.Inc(context.Background(), "downloads", "I do not exist")
	if err != nil {
		t.Errorf("Error trying to increment item from list: %v", err)
	}
	if count != 0 {
		t.Error("Did not properly report non-increment of item.")
	}

	// What about incrementing an item from a list that's not there?
	count, err = s.Inc(context.Background(), "I do not exist", "kernel.tar.gz")
	if err != nil {
		t.Errorf("Error trying to increment item from list: %v", err)
	}
	if count != 0 {
		t.Error("Did not properly report non-increment of item from non-existent list.")
	}
}

func TestAdd(t *testing.T) {
	s := getEmptyStore(t)
	files := []string{"kernel.tar.gz", "vim.tar.gz", "robots.txt"}

	// Does bulk add work?
	count, err := s.Add(context.Background(), "downloads", files...)
	if err != nil {
		t.Errorf("Error bulk inserting: %v", err)
	}
	if count != 3 {
		t.Errorf("Bulk incremented wrong number of items. Expected 5, got %v", count)
	}

	// If we get the list items, do they exist?
	for _, file := range files {
		attempts, ok, err := s.Get(context.Background(), "downloads", file)
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

	// What if we bulk put nothing?
	count, err = s.Add(context.Background(), "downloads", []string{}...)
	if err != nil {
		t.Errorf("Error bulk inserting: %v", err)
	}
	if count != 0 {
		t.Errorf("Bulk added wrong number of items. Expected 0, got %v", count)
	}
}

// These items are expected to be in the db at the start
// of the next few bulk tests.
func bulkAddTestItems(t *testing.T, s *PgStore) {
	// Add a bunch of test items.
	count, err := s.Add(context.Background(), "downloads", "a", "b", "c", "d", "e", "f", "g")
	if err != nil {
		t.Errorf("Error bulk inserting: %v", err)
	}
	if count != 7 {
		t.Errorf("Bulk added wrong number of items. Expected 5, got %v", count)
	}
}

func TestBulkGet(t *testing.T) {
	var tests = []struct {
		afterItem string
		want      []ListEntry
	}{
		{"", []ListEntry{{"a", 0}, {"b", 0}}},
		{"b", []ListEntry{{"c", 0}, {"d", 0}}},
		{"d", []ListEntry{{"e", 0}, {"f", 0}}},
		{"f", []ListEntry{{"g", 0}}},
	}
	s := getEmptyStore(t)
	bulkAddTestItems(t, s)

	// If we bulk get 2 items at a time, does everything work?
	for _, test := range tests {
		items, err := s.BulkGet(context.Background(), "downloads", test.afterItem, 2)
		if err != nil {
			t.Errorf("Error bulk fetching: %v", err)
		}
		if !reflect.DeepEqual(test.want, items) {
			t.Errorf("Expected %v; got %v", test.want, items)
		}
	}

	// What if we bulk get nothing?
	items, err := s.BulkGet(context.Background(), "downloads", "", 0)
	if err != nil {
		t.Errorf("Error bulk deleting: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("Bulk get of nothing yeilded results!")
	}
}

func TestBulkInc(t *testing.T) {
	s := getEmptyStore(t)
	bulkAddTestItems(t, s)

	// Does bulk increment work?
	count, err := s.BulkInc(context.Background(), "downloads", []string{"a", "b", "c", "d", "e"})
	if err != nil {
		t.Errorf("Error bulk incrementing: %v", err)
	}
	if count != 5 {
		t.Errorf("Bulk incremented wrong number of items. Expected 5, got %v", count)
	}

	// If we look for incremented items, are they incremented?
	for _, file := range []string{"a", "b", "c", "d", "e"} {
		attempts, ok, err := s.Get(context.Background(), "downloads", file)
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
		attempts, ok, err := s.Get(context.Background(), "downloads", file)
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

	// What if we bulk increment nothing?
	count, err = s.BulkInc(context.Background(), "downloads", []string{})
	if err != nil {
		t.Errorf("Error bulk deleting: %v", err)
	}
	if count != 0 {
		t.Errorf("Bulk incremented wrong number of items. Expected 0, got %v", count)
	}
}

func TestBulkDel(t *testing.T) {
	s := getEmptyStore(t)
	bulkAddTestItems(t, s)

	// Does bulk delete work?
	count, err := s.BulkDel(context.Background(), "downloads", []string{"a", "b", "c", "d", "e"})
	if err != nil {
		t.Errorf("Error bulk deleting: %v", err)
	}
	if count != 5 {
		t.Errorf("Bulk deleted wrong number of items. Expected 5, got %v", count)
	}

	// If we look for the deleted items, are they correctly missing?
	for _, file := range []string{"a", "b", "c", "d", "e"} {
		_, ok, err := s.Get(context.Background(), "downloads", file)
		if err != nil {
			t.Errorf("Error getting item: %v", err)
		}
		if ok {
			t.Errorf("Found item %v that should have been deleted from list.", file)
		}
	}

	// Were other items left alone?
	for _, file := range []string{"f", "g"} {
		attempts, ok, err := s.Get(context.Background(), "downloads", file)
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

	// What if we bulk delete nothing?
	count, err = s.BulkDel(context.Background(), "downloads", []string{})
	if err != nil {
		t.Errorf("Error bulk deleting: %v", err)
	}
	if count != 0 {
		t.Errorf("Bulk deleted wrong number of items. Expected 0, got %v", count)
	}
}
