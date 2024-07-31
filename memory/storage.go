package memory

import (
	"container/list"
	"slices"
	"sync"

	"github.com/xandalm/go-session"
)

type StorageItem struct {
	id     string
	values map[string]any
}

func (r *StorageItem) Id() string {
	return r.id
}

func (r *StorageItem) Set(k string, v any) {
	r.values[k] = v
}

func (r *StorageItem) Delete(k string) {
	delete(r.values, k)
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
func (s *storage) Save(i session.StorageItem) error {
	if i.Id() == "" {
		panic("empty id")
	}
	item, ok := i.(*StorageItem)
	if !ok {
		panic("unsupported type")
	}
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
		s.idx[pos].anchor.Value = *i
		return nil
	}
	elem := s.items.PushFront(i.values)
	s.idx = slices.Insert(s.idx, pos, &indexNode{
		&i.id,
		elem,
	})
	return nil
}

// Returns the item or an error if can't read from the storage.
func (s *storage) Load(id string) (session.StorageItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	found := s.find(id)
	if found == nil {
		return nil, nil
	}
	return &StorageItem{
		id,
		found.Value.(map[string]any),
	}, nil
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
