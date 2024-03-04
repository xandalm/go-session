package filesystem

import (
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xandalm/go-session/testing/assert"
)

func TestSession_SessionID(t *testing.T) {
	sess := &session{
		id: "abcde",
		v:  map[string]any{},
		ct: time.Now(),
	}

	got := sess.SessionID()
	want := sess.id

	if got != want {
		t.Errorf("got id %q, but want %q", got, want)
	}
}

func TestSession_Get(t *testing.T) {
	sess := &session{
		id: "abcde",
		v:  map[string]any{"key": 123},
		ct: time.Now(),
	}

	t.Run("return 123", func(t *testing.T) {
		got := sess.Get("key")
		want := 123

		if got != want {
			t.Errorf("got value %v, but want %v", got, want)
		}
	})

	t.Run("return nil", func(t *testing.T) {
		got := sess.Get("foo")

		if got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})
}

func TestSession_Set(t *testing.T) {
	sess := &session{
		id: "abcde",
		v:  map[string]any{},
		ct: time.Now(),
	}
	err := sess.Set("foo", "bar")

	assert.NoError(t, err)
	want := "bar"

	if got, ok := sess.v["foo"]; ok {
		if got != want {
			t.Errorf("got value %q, but want %q", got, want)
		}
		return
	}
	t.Error("didn't set value")
}

func TestSession_Delete(t *testing.T) {
	sess := &session{
		id: "abcde",
		v:  map[string]any{"key": 123},
		ct: time.Now(),
	}

	err := sess.Delete("key")

	assert.NoError(t, err)

	if _, ok := sess.v["key"]; ok {
		t.Error("didn't delete value")
	}
}

func TestStorage_CreateSession(t *testing.T) {
	t.Run("create session", func(t *testing.T) {
		path := ""
		dir := "sessions"
		storage := NewStorage(path, dir)

		sid := "abcde"
		got, err := storage.CreateSession(sid)
		assert.NoError(t, err)
		assert.NotNil(t, got)

		sess, ok := got.(*session)
		if !ok {
			t.Fatalf("didn't got session type")
		}
		if sess.id != sid {
			t.Fatalf("got session id %q, but want %q", sess.id, sid)
		}
		if _, err := os.ReadFile(joinPath(path, dir, sid+".sess")); err != nil {
			t.Fatal("cannot open session file")
		}
		t.Cleanup(func() {
			if err := os.RemoveAll(joinPath(path, dir)); err != nil {
				log.Fatalf("didn't complete clean up, %v", err)
			}
		})
	})
}

func joinPath(v ...string) string {
	return filepath.Join(v...)
}
