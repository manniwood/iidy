package iidy

import "testing"

func TestAdd(t *testing.T) {
	m := NewMemStore()
	m.Add("Downloads", "kernel.tar.gz")
	_, ok, _ := m.Get("Downloads", "kernel.tar.gz")
	if !ok {
		t.Error("Did not properly add item to list.")
	}
}

func TestGet(t *testing.T) {
	m := NewMemStore()
	m.Add("Downloads", "kernel.tar.gz")
	_, ok, _ := m.Get("Downloads", "kernel.tar.gz")
	if !ok {
		t.Error("Did not properly get item from list.")
	}
	_, ok, _ = m.Get("Downloads", "I do not exist")
	if ok {
		t.Error("List claims to return value that was not added to list.")
	}
	_, ok, _ = m.Get("I do not exist", "kernel.tar.gz")
	if ok {
		t.Error("Non-existent list claims to return value.")
	}
}

func TestDel(t *testing.T) {
	m := NewMemStore()
	m.Add("Downloads", "kernel.tar.gz")
	err := m.Del("Downloads", "kernel.tar.gz")
	if err != nil {
		t.Error("Error trying to delete item from list: %v", err)
	}
	_, ok, _ := m.Get("Downloads", "kernel.tar.gz")
	if ok {
		t.Error("Did not properly delete item to list.")
	}
}

func TestInc(t *testing.T) {
	m := NewMemStore()
	m.Add("Downloads", "kernel.tar.gz")
	m.Inc("Downloads", "kernel.tar.gz")
	attempts, ok, _ := m.Get("Downloads", "kernel.tar.gz")
	if !ok {
		t.Error("Did not properly add item to list.")
	}
	if attempts != 1 {
		t.Error("Did not properly increment item in list.")
	}
}
