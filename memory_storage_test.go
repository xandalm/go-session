package session

import (
	"testing"
	"time"

	"github.com/xandalm/go-session/testing/assert"
)

func TestMemoryStorage_Save(t *testing.T) {
	t.Run("save session into storage", func(t *testing.T) {

		storage := &MemoryStorage{
			sessions: map[string]Session{},
			list:     []Session{},
		}

		sess := newStubSession("1", time.Now(), dummyFn)

		err := storage.Save(sess)

		assert.NoError(t, err)

		if storage.sessions[sess.Id] != sess {
			t.Errorf("didn't save session")
		}
	})
}

var dummyFn = func(sess Session) error { return nil }

func TestMemoryStorage_Get(t *testing.T) {
	t.Run("restores session from storage", func(t *testing.T) {

		sid := "1"
		sess := newStubSession(sid, time.Now(), dummyFn)
		storage := &MemoryStorage{
			sessions: map[string]Session{sid: sess},
			list:     []Session{sess},
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

func TestMemoryStorage_Rip(t *testing.T) {
	t.Run("remove session from storage", func(t *testing.T) {

		sid := "1"
		sess := newStubSession(sid, time.Now(), dummyFn)
		storage := &MemoryStorage{
			sessions: map[string]Session{sid: sess},
			list:     []Session{sess},
		}

		err := storage.Rip(sid)

		assert.NoError(t, err)

		if _, ok := storage.sessions[sid]; ok {
			t.Error("didn't remove session")
		}
	})
}

func TestMemoryStorage_Reap(t *testing.T) {
	t.Run("remove expired sessions", func(t *testing.T) {
		storage := &MemoryStorage{
			sessions: map[string]Session{},
			list:     []Session{},
		}

		sess1 := newStubSession("1", time.Now(), dummyFn)
		storage.sessions[sess1.Id] = sess1
		storage.list = append(storage.list, sess1)

		sess2 := newStubSession("2", time.Now(), dummyFn)
		storage.sessions[sess2.Id] = sess2
		storage.list = append(storage.list, sess2)

		time.Sleep(time.Microsecond)

		sess3 := newStubSession("3", time.Now(), dummyFn)
		storage.sessions[sess3.Id] = sess3
		storage.list = append(storage.list, sess3)

		storage.Reap(stubAgeChecker(10))

		if len(storage.sessions) > 1 {
			t.Error("didn't remove expired sessions")
		}
	})
}
