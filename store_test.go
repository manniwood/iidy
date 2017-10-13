package iidy

import "testing"

func getEmptyStore(t *testing.T) *PgStore {
	p, err := NewPgStore()
	if err != nil {
		t.Error("Error instantiating PgStore: %v", err)
	}
	p.Nuke()
	return p
}

func TestAdd(t *testing.T) {
	s := getEmptyStore(t)
	s.Add("Downloads", "kernel.tar.gz")
	_, ok, _ := s.Get("Downloads", "kernel.tar.gz")
	if !ok {
		t.Error("Did not properly add item to list.")
	}
}

func TestGet(t *testing.T) {
	s := getEmptyStore(t)
	s.Add("Downloads", "kernel.tar.gz")
	_, ok, _ := s.Get("Downloads", "kernel.tar.gz")
	if !ok {
		t.Error("Did not properly get item from list.")
	}
	_, ok, _ = s.Get("Downloads", "I do not exist")
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
	s.Add("Downloads", "kernel.tar.gz")
	err := s.Del("Downloads", "kernel.tar.gz")
	if err != nil {
		t.Error("Error trying to delete item from list: %v", err)
	}
	_, ok, _ := s.Get("Downloads", "kernel.tar.gz")
	if ok {
		t.Error("Did not properly delete item to list.")
	}
}

func TestInc(t *testing.T) {
	s := getEmptyStore(t)
	s.Add("Downloads", "kernel.tar.gz")
	s.Inc("Downloads", "kernel.tar.gz")
	attempts, ok, _ := s.Get("Downloads", "kernel.tar.gz")
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
	err := s.BulkAdd("Downloads", files)
	if err != nil {
		t.Error("Error bulk inserting: %w", err)
	}
	_, ok, _ := s.Get("Downloads", "kernel.tar.gz")
	if !ok {
		t.Error("Did not properly add item to list.")
	}
	_, ok, _ = s.Get("Downloads", "vim.tar.gz")
	if !ok {
		t.Error("Did not properly add item to list.")
	}
	_, ok, _ = s.Get("Downloads", "robots.txt")
	if !ok {
		t.Error("Did not properly add item to list.")
	}
}
