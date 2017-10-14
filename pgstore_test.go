package iidy

import "testing"

func getEmptyStore(t *testing.T) *PgStore {
	p, err := NewPgStore()
	if err != nil {
		t.Errorf("Error instantiating PgStore: %v", err)
	}
	p.Nuke()
	return p
}

// Our tests add this test item over and over,
// so here it is.
func addTestItem(t *testing.T, s *PgStore) {
	err := s.Add("downloads", "kernel.tar.gz")
	if err != nil {
		t.Errorf("Error adding item: %v", err)
	}
}

func TestAddAndGet(t *testing.T) {
	s := getEmptyStore(t)
	addTestItem(t, s)

	// Did we really add the item?
	attempts, ok, err := s.Get("downloads", "kernel.tar.gz")
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

func TestUnhappyGetScenarios(t *testing.T) {
	s := getEmptyStore(t)

	// What about getting an item that doesn't exist?
	_, ok, err := s.Get("downloads", "I do not exist")
	if err != nil {
		t.Errorf("Error getting item: %v", err)
	}
	if ok {
		t.Error("List claims to return value that was not added to list.")
	}

	// What about getting from a list that doesn't exixt?
	_, ok, err = s.Get("I do not exist", "kernel.tar.gz")
	if err != nil {
		t.Errorf("Error getting item: %v", err)
	}
	if ok {
		t.Error("Non-existent list claims to return value.")
	}
}

func TestDel(t *testing.T) {
	s := getEmptyStore(t)
	addTestItem(t, s)

	// Can we successfully delete?
	err := s.Del("downloads", "kernel.tar.gz")
	if err != nil {
		t.Errorf("Error trying to delete item from list: %v", err)
	}

	// Does getting the deleted value correctly return nothing?
	_, ok, err := s.Get("downloads", "kernel.tar.gz")
	if err != nil {
		t.Errorf("Error getting item: %v", err)
	}
	if ok {
		t.Error("Did not properly delete item to list.")
	}
}

func TestInc(t *testing.T) {
	s := getEmptyStore(t)
	addTestItem(t, s)

	// Does incrementing an item's attempts work?
	err := s.Inc("downloads", "kernel.tar.gz")
	if err != nil {
		t.Errorf("Error trying to increment: %v", err)
	}

	// When we get the incremented attempt, is is correct?
	attempts, ok, err := s.Get("downloads", "kernel.tar.gz")
	if err != nil {
		t.Errorf("Error getting item: %v", err)
	}
	if !ok {
		t.Error("Did not properly add item to list.")
	}
	if attempts != 1 {
		t.Error("Did not properly increment item in list.")
	}
}

func TestBulkAdd(t *testing.T) {
	s := getEmptyStore(t)
	files := []string{"kernel.tar.gz", "vim.tar.gz", "robots.txt"}

	// Does bulk add work?
	err := s.BulkAdd("downloads", files)
	if err != nil {
		t.Errorf("Error bulk inserting: %v", err)
	}

	// If we get the list items, do they exist?
	for _, file := range files {
		attempts, ok, err := s.Get("downloads", file)
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
	files := []string{"a", "b", "c", "d", "e", "f", "g"}
	err := s.BulkAdd("downloads", files)
	if err != nil {
		t.Errorf("Error bulk inserting: %v", err)
	}

	// If we bulk get 2 items at a time, does everything work?
	for _, test := range tests {
		items, err := s.BulkGet("downloads", test.afterItem, 2)
		if err != nil {
			t.Errorf("Error bulk fetching: %v", err)
		}
		if !ListEntrySlicesAreEqual(test.want, items) {
			t.Errorf("Expected %v; got %v", test.want, items)
		}
	}
}

func TestBulkInc(t *testing.T) {
	s := getEmptyStore(t)
	files := []string{"a", "b", "c", "d", "e", "f", "g"}
	err := s.BulkAdd("downloads", files)
	if err != nil {
		t.Errorf("Error bulk inserting: %v", err)
	}

	// Does bulk increment work?
	count, err := s.BulkInc("downloads", []string{"a", "b", "c", "d", "e"})
	if err != nil {
		t.Errorf("Error bulk incrementing: %v", err)
	}
	if count != 5 {
		t.Errorf("Bulk incremented wrong number of items. Expected 5, got %v", count)
	}

	// If we look for incremented items, are they incremented?
	for _, file := range []string{"a", "b", "c", "d", "e"} {
		attempts, ok, err := s.Get("downloads", file)
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
		attempts, ok, err := s.Get("downloads", file)
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
}

func TestBulkDel(t *testing.T) {
	s := getEmptyStore(t)
	files := []string{"a", "b", "c", "d", "e", "f", "g"}
	err := s.BulkAdd("downloads", files)
	if err != nil {
		t.Errorf("Error bulk inserting: %v", err)
	}

	// Does bulk delete work?
	count, err := s.BulkDel("downloads", []string{"a", "b", "c", "d", "e"})
	if err != nil {
		t.Errorf("Error bulk deleting: %v", err)
	}
	if count != 5 {
		t.Errorf("Bulk deleted wrong number of items. Expected 5, got %v", count)
	}

	// If we look for the deleted items, are they correctly missing?
	for _, file := range []string{"a", "b", "c", "d", "e"} {
		_, ok, err := s.Get("downloads", file)
		if err != nil {
			t.Errorf("Error getting item: %v", err)
		}
		if ok {
			t.Errorf("Found item %v that should have been deleted from list.", file)
		}
	}

	// Were other items left alone?
	for _, file := range []string{"f", "g"} {
		attempts, ok, err := s.Get("downloads", file)
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
}

func ListEntrySlicesAreEqual(files []ListEntry, items []ListEntry) bool {
	if files == nil && items == nil {
		return true
	}
	if files == nil || items == nil {
		return false
	}
	if len(files) != len(items) {
		return false
	}
	for i := range files {
		if files[i] != items[i] {
			return false
		}
	}
	return true
}
