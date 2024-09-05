package session

import (
	"fmt"
	"maps"
	"reflect"
	"strings"
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

	cases := []struct {
		key   string
		value any
	}{
		{"string", "abc"},
		{"integer", 1},
		{"float", 1.1},
		{"map", map[string]any{"x": 1, "y": 1}},
		{"slice", []int{1, 2, 3}},
		{"struct", struct{ X, Y int }{1, 1}},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("set %s", strings.ReplaceAll(fmt.Sprintf("%#v", c.value), " ", "")), func(t *testing.T) {
			sess.Set(c.key, c.value)

			got, ok := sess.v[c.key]
			if !ok {
				t.Fatal("didn't set anything")
			}
			if !reflect.DeepEqual(got, c.value) {
				t.Errorf("got value %#v, but want %#v", got, c.value)
			}
		})
	}

	t.Run("panic", func(t *testing.T) {
		cases := []struct {
			desc     string
			value    any
			panicMsg string
		}{
			{
				"when value is a func",
				func() {},
				PanicInvalidByFunc,
			},
			{
				"when value is a pointer",
				new(int),
				PanicInvalidByPointer,
			},
			{
				"when value is a chan",
				make(chan int),
				PanicInvalidByChan,
			},
			{
				"when value is a slice/array and contains func",
				[]any{func() {}},
				PanicInvalidByFunc,
			},
			{
				"when value is a slice/array and contains pointer",
				[]any{new(float32)},
				PanicInvalidByPointer,
			},
			{
				"when value is a slice/array and contains chan",
				[1]any{make(chan struct{})},
				PanicInvalidByChan,
			},
			{
				"when value is a map and can reach in a func value",
				map[string]any{"fn": func() {}},
				PanicInvalidByFunc,
			},
			{
				"when value is a map and can reach in a pointer value",
				map[int]any{1: map[string]any{"p": new(struct{})}},
				PanicInvalidByPointer,
			},
			{
				"when value is a map and can reach in a chan value",
				map[string]any{"ch": make(chan struct{})},
				PanicInvalidByChan,
			},
			{
				"when value is a struct and can reach in a func field",
				struct{ Fn any }{func() {}},
				PanicInvalidByFunc,
			},
			{
				"when value is a struct and can reach in a pointer field",
				struct{ V any }{struct{ P any }{new(struct{})}},
				PanicInvalidByPointer,
			},
			{
				"when value is a struct and can reach in a chan field",
				struct{ Ch any }{make(chan int)},
				PanicInvalidByChan,
			},
		}

		for _, c := range cases {
			t.Run(fmt.Sprintf("set %s", strings.ReplaceAll(fmt.Sprintf("%#v", c.value), " ", "")), func(t *testing.T) {
				defer func() {
					r := recover()
					if r == nil {
						t.Fatal("didn't panic")
					}
					if r != c.panicMsg {
						t.Errorf("panic %s, but want %s", r, c.panicMsg)
					}
				}()
				sess.Set("k", c.value)
			})
		}
	})
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
