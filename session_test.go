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
		Values{},
		nil,
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
			Values{"foo": "bar"},
			nil,
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
		Values{},
		nil,
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
		Values{"foo": "bar"},
		nil,
	}

	sess.Delete("foo")

	if _, ok := sess.v["foo"]; ok {
		t.Error("didn't delete value")
	}

}

func TestSessionFactory(t *testing.T) {
	var sf SessionFactory = DefaultSessionFactory

	assert.NotNil(t, sf)

	t.Run("creates session", func(t *testing.T) {
		id := "1"
		m := Values{"foo": "bar"}

		got := sf.Create(id, m, nil)

		assert.NotNil(t, got)

		sess := got.(*session)
		assert.Equal(t, sess.id, id)
		assert.Equal(t, sess.v, m)

		t.Run("defined meta values can't be mutable by session Set and Delete methods, causing error", func(t *testing.T) {
			sess := sf.Create("1", Values{"foo": "bar"}, nil)

			err := sess.Set("foo", "baz")
			assert.Error(t, err, ErrProtectedKeyName)
		})
	})

	t.Run("restores session", func(t *testing.T) {
		id := "1"
		m := Values{"foo": "bar"}
		v := Values{"baz": "jaz"}

		got := sf.Restore(id, m, v, nil)

		assert.NotNil(t, got)

		sess := got.(*session)
		assert.Equal(t, sess.id, id)

		values := Values{}
		maps.Copy(values, m)
		maps.Copy(values, v)

		assert.Equal(t, sess.v, values)

		t.Run("defined meta values can't be mutable by session Set and Delete methods, causing error", func(t *testing.T) {
			sess := sf.Restore("1", Values{"foo": "bar"}, Values{"baz": "jaz"}, nil)

			err := sess.Set("foo", "baz")
			assert.Error(t, err, ErrProtectedKeyName)
		})

		t.Run("meta values will not be mutable by common values", func(t *testing.T) {
			meta := Values{"foo": "bar"}
			common := Values{"foo": "rab", "baz": "jaz"}

			got := sf.Restore("1", meta, common, nil)

			assert.NotNil(t, got)

			sess := got.(*session)
			assert.Equal(t, sess.id, id)
			assert.Equal(t, sess.v, Values{
				"foo": "bar",
				"baz": "jaz",
			})
		})
	})

	t.Run("override session values", func(t *testing.T) {
		sess := &session{
			id: "1",
			v: Values{
				"update": "before",
				"keep":   "same",
			},
		}

		sf.OverrideValues(
			sess,
			Values{
				"update": "after",
				"new":    "add",
			},
		)

		want := Values{
			"update": "after",
			"new":    "add",
			"keep":   "same",
		}

		assert.Equal(t, sess.v, want)
	})

	t.Run("return session values", func(t *testing.T) {
		sess := &session{
			id: "1",
			v: Values{
				"update": "before",
				"keep":   "same",
			},
		}

		got := sf.ExportValues(sess)
		assert.Equal(t, got, sess.v)
	})
}
