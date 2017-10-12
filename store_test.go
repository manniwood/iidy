package iidy

import "testing"

func getStores(t *testing.T) []Store {
	m := NewMemStore()
	p, err := NewPgStore()
	if err != nil {
		t.Error("Error instantiating PgStore: %v", err)
	}
	stores := make([]Store, 2, 2)
	stores[0] = m
	stores[1] = p
	return stores
}

func TestAdd(t *testing.T) {
	for _, s := range getStores(t) {
		s.Add("Downloads", "kernel.tar.gz")
		_, ok, _ := s.Get("Downloads", "kernel.tar.gz")
		if !ok {
			t.Error("Did not properly add item to list.")
		}
	}
}

func TestGet(t *testing.T) {
	for _, s := range getStores(t) {
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
}

func TestDel(t *testing.T) {
	for _, s := range getStores(t) {
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
}

func TestInc(t *testing.T) {
	for _, s := range getStores(t) {
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
}
