package filesystem

import (
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
