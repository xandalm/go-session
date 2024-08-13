package filesystem

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/xandalm/go-session/testing/assert"
)

func TestStorage_Save(t *testing.T) {
	path := "session_storage_test"
	prefix := "gosess_"
	storage := NewStorage(path, prefix)
	t.Run("create", func(t *testing.T) {
		id := "abcde"

		err := storage.Save(id, map[string]any{"foo": "bar"})

		assert.NoError(t, err)

		file, err := os.Open(filepath.Join(storage.path, fmt.Sprintf("%s%s", prefix, id)))
		if err == nil {
			file.Close()
			return
		}
		t.Error("cannot open session file (was the file created?)")
	})

	t.Cleanup(func() {
		if err := os.RemoveAll(storage.path); err != nil {
			log.Fatalf("cannot clean up after test, %v", err)
		}
	})
}

func TestStorage_Load(t *testing.T) {
	path := "session_storage_test"
	prefix := "gosess_"

	id := "abcde"
	values := map[string]any{
		"foo": "bar",
		"int": 1,
	}

	storage := NewStorage(path, prefix)
	storage.Save(id, values)

	data, err := storage.Read(id)

	assert.NoError(t, err)
	assert.NotNil(t, data)

	assert.Equal(t, data, values)

	t.Cleanup(func() {
		if err := os.RemoveAll(storage.path); err != nil {
			log.Fatalf("cannot clean up after test, %v", err)
		}
	})
}

func TestStorage_List(t *testing.T) {
	path := "session_storage_test"
	prefix := "gosess_"

	id1 := "abcde"
	id2 := "fghij"

	storage := NewStorage(path, prefix)
	storage.Save(id1, map[string]any{})
	storage.Save(id2, map[string]any{})

	got, err := storage.List()

	assert.NoError(t, err)

	if len(got) == 0 {
		t.Fatal("didn't get any name")
	}

	want1 := storage.prefix + id1
	want2 := storage.prefix + id2

	if !slices.Contains(got, want1) || !slices.Contains(got, want2) {
		t.Errorf("unexpected result, %s and %s must be in %v", want1, want2, got)
	}

	t.Cleanup(func() {
		if err := os.RemoveAll(storage.path); err != nil {
			log.Fatalf("cannot clean up after test, %v", err)
		}
	})
}

func TestStorage_Delete(t *testing.T) {
	path := "session_storage_test"
	prefix := "gosess_"

	id := "abcde"
	values := map[string]any{
		"foo": "bar",
		"int": 1,
	}

	storage := NewStorage(path, prefix)
	storage.Save(id, values)

	err := storage.Delete(id)

	assert.NoError(t, err)

	file, err := os.Open(filepath.Join(storage.path, fmt.Sprintf("%s%s", prefix, id)))
	if err == nil {
		file.Close()
		t.Error("the file still exists")
	}

	t.Cleanup(func() {
		if err := os.RemoveAll(storage.path); err != nil {
			log.Fatalf("cannot clean up after test, %v", err)
		}
	})
}
