package session

// import (
// 	"maps"
// 	"reflect"
// 	"testing"
// 	"time"

// 	"github.com/xandalm/go-session/testing/assert"
// )

// func TestMemoryStorage_Save(t *testing.T) {
// 	t.Run("save session into storage", func(t *testing.T) {

// 		storage := &MemoryStorage{
// 			sessions: map[string]registry{},
// 			list:     []registry{},
// 		}

// 		sess := newStubSession("1", time.Now(), dummyFn)

// 		err := storage.Save(sess)

// 		assert.NoError(t, err)

// 		want := registry{
// 			sess.Id,
// 			sess.CreatedAt,
// 			sess.V,
// 		}
// 		if !reflect.DeepEqual(storage.sessions[sess.Id], want) {
// 			t.Errorf("didn't save session")
// 		}
// 	})
// }

// var dummyFn = func(sess Session) error { return nil }

// func TestMemoryStorage_Get(t *testing.T) {
// 	t.Run("call session builder to rebuild session", func(t *testing.T) {
// 		builder := &spySessionBuilder{}
// 		storage := &MemoryStorage{
// 			sessions:       map[string]registry{},
// 			list:           []registry{},
// 			sessionBuilder: builder,
// 		}
// 		sid := "1"
// 		_, reg := createStubSessionAndRegistry(sid, time.Now(), storage.Save)
// 		storage.sessions[sid] = reg
// 		storage.list = append(storage.list, reg)

// 		_, err := storage.Get(sid)

// 		assert.NoError(t, err)

// 		if builder.callsToRestore == 0 {
// 			t.Error("didn't call builder to rebuild")
// 		}
// 	})
// 	t.Run("restores session from storage", func(t *testing.T) {

// 		sid := "1"
// 		storage := &MemoryStorage{
// 			sessions:       map[string]registry{},
// 			list:           []registry{},
// 			sessionBuilder: &stubSessionBuilder{},
// 		}
// 		sess, reg := createStubSessionAndRegistry(sid, time.Now(), storage.Save)
// 		storage.sessions[sid] = reg
// 		storage.list = append(storage.list, reg)

// 		got, err := storage.Get(sid)

// 		assert.NoError(t, err)
// 		assert.NotNil(t, got)

// 		resess, ok := got.(*stubSession)

// 		if !ok {
// 			t.Fatal("didn't get expected session type")
// 		}

// 		checkStubSession(t, resess, sess)
// 	})
// }

// func TestMemoryStorage_Rip(t *testing.T) {
// 	t.Run("remove session from storage", func(t *testing.T) {

// 		storage := &MemoryStorage{
// 			sessions: map[string]registry{},
// 			list:     []registry{},
// 		}
// 		sid := "1"
// 		_, reg := createStubSessionAndRegistry(sid, time.Now(), dummyFn)
// 		storage.sessions[sid] = reg
// 		storage.list = append(storage.list, reg)

// 		err := storage.Rip(sid)

// 		assert.NoError(t, err)

// 		if _, ok := storage.sessions[sid]; ok {
// 			t.Error("didn't remove session")
// 		}
// 	})
// }

// func TestMemoryStorage_Reap(t *testing.T) {
// 	t.Run("remove expired sessions", func(t *testing.T) {
// 		storage := &MemoryStorage{
// 			sessions: map[string]registry{},
// 			list:     []registry{},
// 		}

// 		sess1, reg1 := createStubSessionAndRegistry("1", time.Now(), storage.Save)
// 		storage.sessions[sess1.Id] = reg1
// 		storage.list = append(storage.list, reg1)

// 		sess2, reg2 := createStubSessionAndRegistry("2", time.Now(), storage.Save)
// 		storage.sessions[sess2.Id] = reg2
// 		storage.list = append(storage.list, reg2)

// 		time.Sleep(time.Microsecond)

// 		sess3, reg3 := createStubSessionAndRegistry("3", time.Now(), storage.Save)
// 		storage.sessions[sess3.Id] = reg3
// 		storage.list = append(storage.list, reg3)

// 		storage.Reap(stubNanoAgeChecker(10))

// 		if len(storage.sessions) > 1 {
// 			t.Error("didn't remove expired sessions")
// 		}
// 	})
// }

// func createStubSessionAndRegistry(sid string, t time.Time, fn func(Session) error) (*stubSession, registry) {
// 	sess := newStubSession(sid, t, fn)
// 	reg := newRegistry(sess.SessionID(), sess.CreatedAt, sess.Values())
// 	return sess, reg
// }

// func checkStubSession(t *testing.T, got, want *stubSession) {
// 	t.Helper()

// 	if got.Id != want.Id || !got.CreatedAt.Equal(want.CreatedAt) || !maps.Equal(got.V, want.V) {
// 		t.Errorf("didn't get expected session, got %v but want %v", got, want)
// 	}
// }
