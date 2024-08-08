package memory

import (
	"reflect"
	"testing"

	"github.com/xandalm/go-session/testing/assert"
)

func TestStorage_Save(t *testing.T) {
	storage := NewStorage()
	t.Run("create", func(t *testing.T) {

		err := storage.Save("abcde", map[string]any{"foo": "bar"})

		assert.NoError(t, err)
		assert.NotEmpty(t, storage.items, "didn't store item")
		assert.NotEmpty(t, storage.idx, "was not indexed")
	})
	t.Run("update", func(t *testing.T) {

		err := storage.Save("abcde", map[string]any{"foo": "bar baz"})

		assert.NoError(t, err)

		got := storage.items.Front().Value.(*StorageItem)
		want := StorageItem{
			"abcde",
			map[string]any{"foo": "bar baz"},
		}

		if !reflect.DeepEqual(*got, want) {
			t.Errorf("create %v, but want %v", *got, want)
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
		storage.Save("", map[string]any{})
	})
}

func TestStorage_Load(t *testing.T) {
	id := "abcde"
	item := &StorageItem{"abcde", map[string]any{}}
	storage := NewStorage()

	err := storage.save(item)
	assert.NoError(t, err)
	assert.NotEmpty(t, storage.items)

	got, err := storage.Load(id)

	assert.NoError(t, err)
	assert.NotNil(t, got)

	if !reflect.DeepEqual(item.values, got) {
		t.Errorf("got %v, but want %v", got, item.values)
	}
}

func TestStorage_Delete(t *testing.T) {
	item := &StorageItem{"abcde", map[string]any{}}
	storage := NewStorage()

	err := storage.save(item)
	assert.NoError(t, err)
	assert.NotEmpty(t, storage.items)

	err = storage.Delete(item.id)

	assert.NoError(t, err)
	assert.NotEmpty(t, storage.items, "didn't delete")
}
