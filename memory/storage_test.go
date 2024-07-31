package memory

import (
	"reflect"
	"testing"

	"github.com/xandalm/go-session/testing/assert"
)

func TestStorage_Save(t *testing.T) {
	storage := NewStorage()
	t.Run("create", func(t *testing.T) {
		item := &StorageItem{
			"abcde",
			map[string]any{"foo": "bar"},
		}

		err := storage.Save(item)

		assert.Nil(t, err)
		assert.NotEmpty(t, storage.items, "didn't store item")
		assert.NotEmpty(t, storage.idx, "was not indexed")
	})
	t.Run("update", func(t *testing.T) {
		item := &StorageItem{
			"abcde",
			map[string]any{"foo": "bar baz"},
		}

		err := storage.Save(item)

		assert.Nil(t, err)
		got := storage.items.Front().Value.(StorageItem)
		if !reflect.DeepEqual(got, *item) {
			t.Errorf("create %v, but want %v", got, item.values)
		}
	})
	t.Run("panic on empty id", func(t *testing.T) {
		storage := NewStorage()
		defer func() {
			r := recover()
			if r == nil || r != "empty id" {
				t.Error("didn't panic")
			}
		}()
		storage.Save(&StorageItem{})
	})
}

func TestStorage_Load(t *testing.T) {
	id := "abcde"
	item := &StorageItem{"abcde", map[string]any{}}
	storage := NewStorage()

	err := storage.save(item)
	assert.Nil(t, err)
	assert.NotEmpty(t, storage.items)

	got, err := storage.Load(id)

	assert.Nil(t, err)
	assert.NotNil(t, got)

	if !reflect.DeepEqual(item, got) {
		t.Errorf("got %v, but want %v", got, item)
	}
}

func TestStorage_Delete(t *testing.T) {
	item := &StorageItem{"abcde", map[string]any{}}
	storage := NewStorage()

	err := storage.save(item)
	assert.Nil(t, err)
	assert.NotEmpty(t, storage.items)

	err = storage.Delete(item.id)

	assert.Nil(t, err)
	assert.NotEmpty(t, storage.items, "didn't delete")
}
