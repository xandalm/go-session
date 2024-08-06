package session

import (
	"testing"

	"github.com/xandalm/go-session/testing/assert"
)

func TestSession_SessionID(t *testing.T) {
	sess := &session{
		nil,
		"abcde",
		map[string]any{},
		NowTimeNanoseconds(),
		NowTimeNanoseconds(),
	}

	got := sess.SessionID()
	want := "abcde"

	if got != want {
		t.Errorf("got id %q, but want %q", got, want)
	}
}

func TestSession_Get(t *testing.T) {
	t.Run("return value", func(t *testing.T) {
		dummyProvider := &stubProvider{}
		sess := &session{
			dummyProvider,
			"abcde",
			map[string]any{"foo": "bar"},
			NowTimeNanoseconds(),
			NowTimeNanoseconds(),
		}

		got := sess.Get("foo")
		want := "bar"

		if got != want {
			t.Errorf("got value %q, but want %q", got, want)
		}
	})

	t.Run("tell provider to sync data", func(t *testing.T) {
		provider := &spyProvider{}

		(&session{
			p:  provider,
			id: "abcde",
		}).Get("foo")

		if provider.callsToSync == 0 {
			t.Fatal("didn't tell provider")
		}
	})
}

func TestSession_Set(t *testing.T) {
	sess := &session{
		nil,
		"abcde",
		map[string]any{},
		NowTimeNanoseconds(),
		NowTimeNanoseconds(),
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
		nil,
		"abcde",
		map[string]any{"foo": "bar"},
		NowTimeNanoseconds(),
		NowTimeNanoseconds(),
	}

	err := sess.Delete("foo")

	assert.Nil(t, err)

	if _, ok := sess.v["foo"]; ok {
		t.Error("didn't delete value")
	}
}
