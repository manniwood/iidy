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

func TestAdd(t *testing.T) {
	s := getEmptyStore(t)
	s.Add("downloads", "kernel.tar.gz")
	_, ok, _ := s.Get("downloads", "kernel.tar.gz")
	if !ok {
		t.Error("Did not properly add item to list.")
	}
}

func TestGet(t *testing.T) {
	s := getEmptyStore(t)
	s.Add("downloads", "kernel.tar.gz")
	_, ok, _ := s.Get("downloads", "kernel.tar.gz")
	if !ok {
		t.Error("Did not properly get item from list.")
	}
	_, ok, _ = s.Get("downloads", "I do not exist")
	if ok {
		t.Error("List claims to return value that was not added to list.")
	}
	_, ok, _ = s.Get("I do not exist", "kernel.tar.gz")
	if ok {
		t.Error("Non-existent list claims to return value.")
	}
}

func TestDel(t *testing.T) {
	s := getEmptyStore(t)
	s.Add("downloads", "kernel.tar.gz")
	err := s.Del("downloads", "kernel.tar.gz")
	if err != nil {
		t.Errorf("Error trying to delete item from list: %v", err)
	}
	_, ok, _ := s.Get("downloads", "kernel.tar.gz")
	if ok {
		t.Error("Did not properly delete item to list.")
	}
}

func TestInc(t *testing.T) {
	s := getEmptyStore(t)
	s.Add("downloads", "kernel.tar.gz")
	s.Inc("downloads", "kernel.tar.gz")
	attempts, ok, _ := s.Get("downloads", "kernel.tar.gz")
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
	err := s.BulkAdd("downloads", files)
	if err != nil {
		t.Errorf("Error bulk inserting: %w", err)
	}
	for _, file := range files {
		attempts, ok, _ := s.Get("downloads", file)
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
		startKey string
		want     []ListItem
	}{
		{"", []ListItem{{"a", 0}, {"b", 0}}},
		{"b", []ListItem{{"c", 0}, {"d", 0}}},
		{"d", []ListItem{{"e", 0}, {"f", 0}}},
		{"f", []ListItem{{"g", 0}}},
	}
	s := getEmptyStore(t)
	files := []string{"a", "b", "c", "d", "e", "f", "g"}
	err := s.BulkAdd("downloads", files)
	if err != nil {
		t.Errorf("Error bulk inserting: %w", err)
	}

	for _, test := range tests {
		items, err := s.BulkGet("downloads", test.startKey, 2)
		if err != nil {
			t.Errorf("Error bulk fetching: %v", err)
		}
		if !ItemSlicesAreEqual(test.want, items) {
			t.Errorf("Expected %v; got %v", test.want, items)
		}
	}
}

func ItemSlicesAreEqual(files []ListItem, items []ListItem) bool {
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

func TestBulkInc(t *testing.T) {
	s := getEmptyStore(t)
	files := []string{"a", "b", "c", "d", "e", "f", "g"}
	err := s.BulkAdd("downloads", files)
	if err != nil {
		t.Errorf("Error bulk inserting: %w", err)
	}
	count, err := s.BulkInc("downloads", []string{"a", "b", "c", "d", "e"})
	if err != nil {
		t.Errorf("Error bulk incrementing: %w", err)
	}
	if count != 5 {
		t.Errorf("Bulk incremented wrong number of items. Expected 5, got %v", count)
	}
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
		t.Errorf("Error bulk inserting: %w", err)
	}
	count, err := s.BulkDel("downloads", []string{"a", "b", "c", "d", "e"})
	if err != nil {
		t.Errorf("Error bulk deleting: %w", err)
	}
	if count != 5 {
		t.Errorf("Bulk deleted wrong number of items. Expected 5, got %v", count)
	}
	for _, file := range []string{"a", "b", "c", "d", "e"} {
		_, ok, err := s.Get("downloads", file)
		if err != nil {
			t.Errorf("Error getting item: %v", err)
		}
		if ok {
			t.Errorf("Found item %v that should have been deleted from list.", file)
		}
	}
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
