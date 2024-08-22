package session

import (
	"maps"
	"sync"
	"testing"

	"github.com/xandalm/go-session/testing/assert"
)

func TestSession_SessionID(t *testing.T) {
	sess := &session{
		sync.Mutex{},
		"abcde",
		map[string]any{},
	}

	got := sess.SessionID()
	want := "abcde"

	if got != want {
		t.Errorf("got id %q, but want %q", got, want)
	}
}

func TestSession_Get(t *testing.T) {
	t.Run("return value", func(t *testing.T) {
		sess := &session{
			sync.Mutex{},
			"abcde",
			map[string]any{"foo": "bar"},
		}

		got := sess.Get("foo")
		want := "bar"

		if got != want {
			t.Errorf("got value %q, but want %q", got, want)
		}
	})

}

func TestSession_Set(t *testing.T) {
	sess := &session{
		sync.Mutex{},
		"abcde",
		map[string]any{},
	}
	key := "foo"
	value := "bar"

	sess.Set(key, value)

	got, ok := sess.v[key]
	if !ok {
		t.Fatal("didn't set anything")
	}
	if got != value {
		t.Errorf("got value %q, but want %q", got, value)
	}
}

func TestSession_Delete(t *testing.T) {
	sess := &session{
		sync.Mutex{},
		"abcde",
		map[string]any{"foo": "bar"},
	}

	sess.Delete("foo")

	if _, ok := sess.v["foo"]; ok {
		t.Error("didn't delete value")
	}

}

func TestSessionFactory(t *testing.T) {
	var sf SessionFactory = NewSessionFactory()

	assert.NotNil(t, sf)

	t.Run("creates session", func(t *testing.T) {
		id := "1"
		m := map[string]any{"foo": "bar"}

		got := sf.Create(id, m)

		assert.NotNil(t, got)

		sess := got.(*session)
		assert.Equal(t, sess.id, id)
		assert.Equal(t, sess.v, m)

		t.Run("defined meta values can't be mutable by session Set and Delete methods, causing error", func(t *testing.T) {
			sess := sf.Create("1", map[string]any{"foo": "bar"})

			err := sess.Set("foo", "baz")
			assert.Error(t, err, ErrProtectedKeyName)
		})
	})

	t.Run("restores session", func(t *testing.T) {
		id := "1"
		m := map[string]any{"foo": "bar"}
		v := map[string]any{"baz": "jaz"}

		got := sf.Restore(id, m, v)

		assert.NotNil(t, got)

		sess := got.(*session)
		assert.Equal(t, sess.id, id)

		values := map[string]any{}
		maps.Copy(values, m)
		maps.Copy(values, v)

		assert.Equal(t, sess.v, values)

		t.Run("defined meta values can't be mutable by session Set and Delete methods, causing error", func(t *testing.T) {
			sess := sf.Restore("1", map[string]any{"foo": "bar"}, map[string]any{"baz": "jaz"})

			err := sess.Set("foo", "baz")
			assert.Error(t, err, ErrProtectedKeyName)
		})
	})
}
