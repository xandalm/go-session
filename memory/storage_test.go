package memory

import (
	"reflect"
	"slices"
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

func TestStorage_Read(t *testing.T) {
	id := "abcde"
	item := &StorageItem{"abcde", map[string]any{}}
	storage := NewStorage()

	err := storage.save(item)
	assert.NoError(t, err)
	assert.NotEmpty(t, storage.items)

	got, err := storage.Read(id)

	assert.NoError(t, err)
	assert.NotNil(t, got)

	if !reflect.DeepEqual(item.values, got) {
		t.Errorf("got %v, but want %v", got, item.values)
	}
}

func TestStorage_List(t *testing.T) {
	id1 := "abcde"
	id2 := "fghij"
	storage := NewStorage()

	assert.NoError(t, storage.save(&StorageItem{id1, map[string]any{}}))
	assert.NoError(t, storage.save(&StorageItem{id2, map[string]any{}}))
	assert.NotEmpty(t, storage.items)

	got, err := storage.List()

	assert.NoError(t, err)
	assert.NotNil(t, got)

	if len(got) == 0 {
		t.Fatal("didn't get any name")
	}

	if !slices.Contains(got, id1) || !slices.Contains(got, id2) {
		t.Errorf("unexpected result, %s and %s must be in %v", id1, id2, got)
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
