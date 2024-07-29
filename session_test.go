package session

import (
	"testing"
	"time"

	"github.com/xandalm/go-session/testing/assert"
)

func TestSession_SessionID(t *testing.T) {
	sess := &session{
		"abcde",
		map[string]any{},
		time.Now(),
		time.Now(),
	}

	got := sess.SessionID()
	want := "abcde"

	if got != want {
		t.Errorf("got id %q, but want %q", got, want)
	}
}

func TestSession_Get(t *testing.T) {
	sess := &session{
		"abcde",
		map[string]any{"foo": "bar"},
		time.Now(),
		time.Now(),
	}

	got := sess.Get("foo")
	want := "bar"

	if got != want {
		t.Errorf("got value %q, but want %q", got, want)
	}
}

func TestSession_Set(t *testing.T) {
	sess := &session{
		"abcde",
		map[string]any{},
		time.Now(),
		time.Now(),
	}
	key := "foo"
	value := "bar"

	err := sess.Set(key, value)

	assert.Nil(t, err)

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
		"abcde",
		map[string]any{"foo": "bar"},
		time.Now(),
		time.Now(),
	}

	err := sess.Delete("foo")

	assert.Nil(t, err)

	if _, ok := sess.v["foo"]; ok {
		t.Error("didn't delete value")
	}
}
