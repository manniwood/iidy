package iidy

import (
	"fmt"

	"github.com/pkg/errors"
)

type MemStore struct {
	// map[list name]map[list item ID]number of attempts
	store map[string]map[string]uint
}

func NewMemStore() Store {
	m := make(map[string]map[string]uint)
	s := MemStore{store: m}
	return &s
}

func (m *MemStore) Add(listName string, itemID string) (err error) {
	list, ok := m.store[listName]
	if !ok {
		list = make(map[string]uint)
		m.store[listName] = list
	}
	list[itemID] = 0
	return nil
}

func (m *MemStore) Get(listName string, itemID string) (attempts uint, ok bool, err error) {
	list, ok := m.store[listName]
	if !ok {
		return 0, false, nil
	}
	item, ok := list[itemID]
	return item, ok, nil
}

func (m *MemStore) Del(listName string, itemID string) (err error) {
	list, ok := m.store[listName]
	if !ok {
		return nil
	}
	delete(list, itemID)
	return nil
}

func (m *MemStore) Inc(listName string, itemID string) (err error) {
	list, ok := m.store[listName]
	if !ok {
		return errors.New(fmt.Sprintf("List %s does not exist", listName))
	}
	if _, ok := list[itemID]; !ok {
		return errors.New(fmt.Sprintf("In list %s, item ID %s does not exist", listName, itemID))
	}
	list[itemID]++
	return nil
}
