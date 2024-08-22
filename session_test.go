package session

import (
	"sync"
	"testing"

	"github.com/xandalm/go-session/testing/assert"
)

func TestSession_SessionID(t *testing.T) {
	sess := &session{
		sync.Mutex{},
		// nil,
		"abcde",
		map[string]any{},
		NowTimeNanoseconds(),
		NowTimeNanoseconds(),
		true,
	}

	got := sess.SessionID()
	want := "abcde"

	if got != want {
		t.Errorf("got id %q, but want %q", got, want)
	}
}

func TestSession_Get(t *testing.T) {
	t.Run("return value", func(t *testing.T) {
		// dummyProvider := &stubProvider{}
		sess := &session{
			sync.Mutex{},
			// dummyProvider,
			"abcde",
			map[string]any{"foo": "bar"},
			NowTimeNanoseconds(),
			NowTimeNanoseconds(),
			true,
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
		// nil,
		"abcde",
		map[string]any{},
		NowTimeNanoseconds(),
		NowTimeNanoseconds(),
		true,
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
		// nil,
		"abcde",
		map[string]any{"foo": "bar"},
		NowTimeNanoseconds(),
		NowTimeNanoseconds(),
		true,
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
		v := map[string]any{"foo": "bar"}

		got := sf.Create(id, v)

		assert.NotNil(t, got)

		sess := got.(*session)
		assert.Equal(t, sess.id, id)
		assert.Equal(t, sess.v, v)
	})
}
