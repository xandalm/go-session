package memory

import (
	"container/list"
	"slices"
	"sync"
)

type StorageItem struct {
	id     string
	values map[string]any
}

func (i *StorageItem) Id() string {
	return i.id
}

func (i *StorageItem) Set(k string, v any) {
	i.values[k] = v
}

func (i *StorageItem) Delete(k string) {
	delete(i.values, k)
}

func (i *StorageItem) Values() map[string]any {
	return i.values
}

type indexNode struct {
	id     *string
	anchor *list.Element
}

type index []*indexNode

type storage struct {
	mu    sync.Mutex
	items *list.List
	idx   index
}

func NewStorage() *storage {
	return &storage{
		items: list.New(),
		idx:   index{},
	}
}

// Save item into storage.
func (s *storage) Save(id string, values map[string]any) error {
	if id == "" {
		panic("empty id")
	}
	item := &StorageItem{id, values}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.save(item); err != nil {
		return err
	}
	return nil
}

func (s *storage) getPos(id string) (int, bool) {
	return slices.BinarySearchFunc(s.idx, id, func(e *indexNode, s string) int {
		if *e.id < s {
			return -1
		}
		if *e.id > s {
			return 1
		}
		return 0
	})
}

func (s *storage) find(id string) *list.Element {
	pos, has := s.getPos(id)
	if has {
		return s.idx[pos].anchor
	}
	return nil
}

func (s *storage) save(i *StorageItem) error {
	pos, has := s.getPos(i.id)
	if has {
		s.idx[pos].anchor.Value = i
		return nil
	}
	elem := s.items.PushFront(i)
	s.idx = slices.Insert(s.idx, pos, &indexNode{
		&i.id,
		elem,
	})
	return nil
}

// Returns the item or an error if can't read from the storage.
func (s *storage) Load(id string) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	found := s.find(id)
	if found == nil {
		return nil, nil
	}
	return found.Value.(*StorageItem).values, nil
}

// Delete item from the storage.
func (s *storage) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	pos, has := s.getPos(id)
	if has {
		s.items.Remove(s.idx[pos].anchor)
		s.idx = slices.Delete(s.idx, pos, pos+1)
		return nil
	}
	return nil
}
