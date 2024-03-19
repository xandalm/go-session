package filesystem_test

import (
	"log"
	"os"
	"testing"
	"time"

	"github.com/xandalm/go-session"
	"github.com/xandalm/go-session/filesystem"
	"github.com/xandalm/go-session/testing/assert"
)

func TestSessionLifecycleAtStorage(t *testing.T) {
	path := "sessions_from_acceptance_test"
	storage := filesystem.Storage(path)
	sid := "a63140d2bb051e439c790a4d35c0"

	t.Run("create session", func(t *testing.T) {
		sess, err := storage.CreateSession(sid)

		assert.NoError(t, err)
		assert.NotNil(t, sess)

		if sess.SessionID() != sid {
			t.Errorf("got session with id %s, but want id %s", sess.SessionID(), sid)
		}
	})
	t.Run("get session", func(t *testing.T) {
		sess, err := storage.GetSession(sid)

		assert.NoError(t, err)
		assert.NotNil(t, sess)

		if sess.SessionID() != sid {
			t.Errorf("got session with id %s, but want id %s", sess.SessionID(), sid)
		}
	})
	// reread session from the file
	sess, _ := storage.GetSession(sid)
	key := "username"
	value := "xandalm"

	t.Run("set session value", func(t *testing.T) {
		err := sess.Set(key, value)
		assert.NoError(t, err)
	})
	t.Run("get session value", func(t *testing.T) {
		gotValue := sess.Get(key)
		assert.NotNil(t, gotValue)
		assert.Equal(t, value, gotValue.(string), "expected to get value %q, but got %q", value, gotValue.(string))
	})
	// reread session from the file
	sess, _ = storage.GetSession(sess.SessionID())
	t.Run("the session file was updated after set value", func(t *testing.T) {
		gotValue := sess.Get(key)
		assert.Equal(t, value, gotValue.(string), "expected to get value %q, but got %q", value, gotValue.(string))
	})
	t.Run("delete session value", func(t *testing.T) {
		err := sess.Delete(key)
		assert.NoError(t, err)
		assert.Nil(t, sess.Get(key), "didn't delete session value")
	})
	// reread session from the file
	sess, _ = storage.GetSession(sess.SessionID())
	t.Run("the session file was updated after delete value", func(t *testing.T) {
		assert.Nil(t, sess.Get(key), "didn't delete session value")
	})

	t.Run("reap session", func(t *testing.T) {
		err := storage.ReapSession(sess.SessionID())

		assert.NoError(t, err)
		got, err := storage.GetSession(sess.SessionID())
		assert.NoError(t, err)
		assert.Nil(t, got, "didn't reap session")
	})
	t.Run("deadline for expired sessions", func(t *testing.T) {
		sess, err := storage.CreateSession(sid)
		assert.NoError(t, err)
		assert.NotNil(t, sess)

		time.Sleep(time.Second + 1)

		storage.Deadline(session.SecondsAgeCheckerAdapter(1))

		sess, err = storage.GetSession(sess.SessionID())
		assert.NoError(t, err)
		assert.Nil(t, sess, "didn't remove expired session")
	})
	t.Cleanup(func() {
		if err := os.RemoveAll(path); err != nil {
			log.Fatalf("cannot clean up after test, %v", err)
		}
	})
}
