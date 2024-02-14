package session

import (
	"testing"
	"time"

	"github.com/xandalm/go-session/testing/assert"
)

func TestMemoryStorage_Save(t *testing.T) {
	t.Run("save session into storage", func(t *testing.T) {
		storage := &MemoryStorage{
			sessions: map[string]ISession{},
		}

		sess := newStubSession("1", time.Now(), dummyFn)

		err := storage.Save(sess)

		assert.NoError(t, err)

		if storage.sessions[sess.Id] != sess {
			t.Errorf("didn't save session")
		}
	})
}

var dummyFn = func(sess ISession) error { return nil }

func TestMemoryStorage_Get(t *testing.T) {
	t.Run("restores session from storage", func(t *testing.T) {

		sid := "1"
		sess := newStubSession(sid, time.Now(), dummyFn)
		storage := &MemoryStorage{
			sessions: map[string]ISession{
				sid: sess,
			},
		}

		got, err := storage.Get(sid)

		assert.NoError(t, err)
		assert.NotNil(t, got)

		resess, ok := got.(*stubSession)

		if !ok {
			t.Fatal("didn't get expected session type")
		}

		assert.Equal(t, resess, sess)
	})
}
