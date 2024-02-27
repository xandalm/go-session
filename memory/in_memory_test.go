package memory

import (
	"reflect"
	"testing"
	"time"

	"github.com/xandalm/go-session/testing/assert"
)

func TestSession_SessionID(t *testing.T) {
	sess := &session{
		"abcde",
		map[string]any{},
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
	}
	key := "foo"
	value := "bar"

	err := sess.Set(key, value)

	assert.NoError(t, err)

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
	}

	err := sess.Delete("foo")

	assert.NoError(t, err)

	if _, ok := sess.v["foo"]; ok {
		t.Error("didn't delete value")
	}
}

func TestStorage_CreateSession(t *testing.T) {
	t.Run("create session", func(t *testing.T) {
		storage := &storage{sessions: map[string]*session{}}
		sid := "abcde"

		sess, err := storage.CreateSession(sid)

		assert.NoError(t, err)
		assert.NotNil(t, sess)

		_, ok := storage.sessions[sid]
		if !ok {
			t.Fatal("didn't create session")
		}
	})
	t.Run("panic on empty session id", func(t *testing.T) {
		storage := &storage{sessions: map[string]*session{}}
		defer func() {
			r := recover()
			if r == nil || r != "session: empty sid (session id)" {
				t.Error("didn't panic")
			}
		}()
		storage.CreateSession("")
	})
}

func TestStorage_GetSession(t *testing.T) {
	sid := "abcde"
	sess := newSession(sid)
	storage := &storage{
		sessions: map[string]*session{
			sess.id: sess,
		},
	}

	got, err := storage.GetSession(sid)

	assert.NoError(t, err)
	assert.NotNil(t, got)

	_sess := got.(*session)
	if !reflect.DeepEqual(sess, _sess) {
		t.Errorf("got session %v, but want %v", _sess, sess)
	}
}

func TestStorage_ReapSession(t *testing.T) {
	sid := "abcde"
	sess := newSession(sid)
	storage := &storage{
		sessions: map[string]*session{
			sess.id: sess,
		},
	}

	err := storage.ReapSession(sid)

	assert.NoError(t, err)

	if _, ok := storage.sessions[sid]; ok {
		t.Error("didn't remove session")
	}
}

func TestStorage_Deadline(t *testing.T) {

	storage := &storage{sessions: map[string]*session{}}

	sess1 := newSession("abcde")
	storage.sessions[sess1.id] = sess1

	sess2 := newSession("fghij")
	storage.sessions[sess2.id] = sess2

	time.Sleep(time.Millisecond)

	sess3 := newSession("klmno")
	storage.sessions[sess3.id] = sess3

	checker := stubMilliAgeChecker(1)

	storage.Deadline(checker)

	if len(storage.sessions) > 1 {
		t.Fatal("didn't remove expired sessions")
	}

	got, ok := storage.sessions[sess3.id]
	if !ok {
		t.Errorf("expected session %q, but got %q", sess3.id, got.id)
	}
}

type stubMilliAgeChecker int64

func (c stubMilliAgeChecker) ShouldReap(t time.Time) bool {
	return time.Now().UnixMilli()-t.UnixMilli() >= int64(c)
}
