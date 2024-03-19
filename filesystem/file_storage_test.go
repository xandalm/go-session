package filesystem

import (
	"container/list"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/xandalm/go-session/testing/assert"
)

func TestGetSessionID(t *testing.T) {
	sess := &session{
		id: "abcde",
		v:  map[string]any{},
		ct: time.Now(),
	}

	got := sess.SessionID()
	want := sess.id

	if got != want {
		t.Errorf("got id %q, but want %q", got, want)
	}
}

func TestGetValueFromSession(t *testing.T) {
	sess := &session{
		id: "abcde",
		v:  map[string]any{"key": 123},
		ct: time.Now(),
	}

	t.Run("return 123", func(t *testing.T) {
		got := sess.Get("key")
		want := 123

		if got != want {
			t.Errorf("got value %v, but want %v", got, want)
		}
	})

	t.Run("return nil", func(t *testing.T) {
		got := sess.Get("foo")

		if got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})
}

func TestSetValueFromSession(t *testing.T) {

	sess := &session{
		id: "abcde",
		v:  map[string]any{},
		ct: time.Now(),
	}
	io := &stubStorageIO{regs: map[string]*extSession{
		sess.id: {
			sess.v,
			sess.ct.UnixNano(),
			sess.at.UnixNano(),
		},
	}}
	_storage.io = io
	t.Cleanup(func() {
		_storage.io = _io // default io
	})

	cases := []struct {
		typ   string
		value any
		want  any
	}{
		{"string", "bar", "bar"},
		{"int", 1, 1},
		{"struct", struct{ Id int }{10}, map[string]any{"Id": 10}},
		{"map[string]int", map[string]int{"a": 1}, map[string]any{"a": 1}},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("set %s value", c.typ), func(t *testing.T) {
			err := sess.Set("foo", c.value)

			assert.NoError(t, err)

			if got, ok := sess.v["foo"]; ok {
				if !reflect.DeepEqual(got, c.want) {
					t.Errorf("set value to %v, but want %v", got, c.want)
				}
				got, ok := io.regs[sess.id].V["foo"]
				if !ok {
					t.Error("didn't update storage")
				}
				if !reflect.DeepEqual(got, c.want) {
					t.Errorf("storage is updated to wrong value, got %v want %v", got, c.want)
				}
				return
			}
			t.Error("didn't set value")
		})
	}
	t.Run("panic when try to set a func", func(t *testing.T) {
		defer func() {
			r := recover()
			if r == nil || r != "session: cannot stores func into session" {
				t.Errorf("didn't get expected panic, got %v", r)
			}
		}()
		sess.Set("foo", func() any { return "foo" })
	})
	t.Run("panic when try to set a chan", func(t *testing.T) {
		defer func() {
			r := recover()
			if r == nil || r != "session: cannot stores chan into session" {
				t.Errorf("didn't get expected panic, got %v", r)
			}
		}()
		sess.Set("foo", make(chan int))
	})
}

func TestDeleteValueFromSession(t *testing.T) {

	sess := &session{
		id: "abcde",
		v:  map[string]any{"key": 123},
		ct: time.Now(),
		at: time.Now(),
	}
	io := &stubStorageIO{regs: map[string]*extSession{
		sess.id: {
			sess.v,
			sess.ct.UnixNano(),
			sess.at.UnixNano(),
		},
	}}
	_storage.io = io
	t.Cleanup(func() {
		_storage.io = _io // default io
	})
	err := sess.Delete("key")

	assert.NoError(t, err)

	if _, ok := sess.v["key"]; ok {
		t.Error("didn't delete value")
	}
	if _, ok := io.regs[sess.id].V["key"]; ok {
		t.Error("didn't update storage, value still exists")
	}
}

type stubStorageIO struct {
	regs map[string]*extSession
}

func (sio *stubStorageIO) Create(sid string) (*session, error) {
	now := time.Now()
	sess := &session{sid, map[string]any{}, now, now}

	sio.regs[sid] = &extSession{
		sess.v,
		sess.ct.UnixNano(),
		sess.at.UnixNano(),
	}

	return sess, nil
}

func (sio *stubStorageIO) Read(sid string) (*session, error) {
	if reg, ok := sio.regs[sid]; ok {
		return &session{
			sid,
			reg.V,
			time.Unix(0, reg.Ct),
			time.Unix(0, reg.At),
		}, nil
	}

	return nil, nil
}

func (sio *stubStorageIO) Write(sess *session) error {
	esess := sio.regs[sess.id]
	esess.At = time.Now().UnixNano()
	esess.V = sess.v
	sio.regs[sess.id] = esess
	return nil
}

func (sio *stubStorageIO) Delete(sid string) error {
	delete(sio.regs, sid)
	return nil
}

func (sio *stubStorageIO) List() []string {
	names := make([]string, len(sio.regs))
	var x, y int
	for name, reg := range sio.regs {
		for x = y - 1; x >= 0; x-- {
			if reg.Ct >= sio.regs[names[x]].Ct {
				break
			}
			names[x+1] = names[x]
		}
		names[x+1] = name
		y++
	}
	return names
}

var dummyMap = map[string]*list.Element{}
var dummyList = list.New()

func TestCreatingSessionInStorage(t *testing.T) {
	t.Run("create session", func(t *testing.T) {
		io := &stubStorageIO{regs: map[string]*extSession{}}
		storage := &storage{
			io,
			dummyMap,
			dummyList,
			sync.Mutex{},
		}

		sid := "abcde"
		got, err := storage.CreateSession(sid)
		assert.NoError(t, err)
		assert.NotNil(t, got)

		sess, ok := got.(*session)
		if !ok {
			t.Fatalf("didn't got session type")
		}
		if sess.id != sid {
			t.Fatalf("got session id %q, but want %q", sess.id, sid)
		}
		if _, ok := io.regs[sid]; !ok {
			t.Errorf("didn't write session")
		}
	})
}

func TestGettingSessionFromStorage(t *testing.T) {

	t.Run("returns session", func(t *testing.T) {

		sid := "abcde"

		sess := &session{
			sid,
			map[string]any{},
			time.Now(),
			time.Now(),
		}
		m, l := createSessionsMapAndList(sess)
		storage := &storage{
			io: &stubStorageIO{
				regs: map[string]*extSession{
					sid: createExtSessionFromSession(sess),
				},
			},
			m:    m,
			list: l,
		}

		got, err := storage.GetSession(sid)

		assert.NoError(t, err)
		assert.NotNil(t, got)

		sess, ok := got.(*session)
		if !ok {
			t.Fatalf("didn't got session type")
		}
		if sess.id != sid {
			t.Fatalf("got session id %q, but want %q", sess.id, sid)
		}
	})
}

func TestReapingSessionFromStorage(t *testing.T) {
	t.Run("removes session", func(t *testing.T) {

		sid := "abcde"

		sess := &session{
			sid,
			map[string]any{},
			time.Now(),
			time.Now(),
		}
		m, l := createSessionsMapAndList(sess)
		io := &stubStorageIO{
			regs: map[string]*extSession{
				sid: createExtSessionFromSession(sess),
			},
		}
		storage := &storage{
			io:   io,
			m:    m,
			list: l,
		}

		err := storage.ReapSession(sid)

		assert.NoError(t, err)

		if _, ok := io.regs[sid]; ok {
			t.Error("didn't remove session")
		}
	})
}

func TestDeadlineCheckUpInStorage(t *testing.T) {
	t.Run("remove expired session", func(t *testing.T) {

		regs := map[string]*extSession{}
		sess1 := &session{"1", map[string]any{}, time.Now(), time.Now()}
		regs[sess1.id] = createExtSessionFromSession(sess1)
		sess2 := &session{"2", map[string]any{}, time.Now(), time.Now()}
		regs[sess2.id] = createExtSessionFromSession(sess2)

		time.Sleep(10 * time.Millisecond)

		sess3 := &session{"3", map[string]any{}, time.Now(), time.Now()}
		regs[sess3.id] = createExtSessionFromSession(sess3)

		io := &stubStorageIO{regs: regs}
		m, l := createSessionsMapAndList(sess1, sess2, sess3)
		storage := &storage{io, m, l, sync.Mutex{}}

		storage.Deadline(stubMilliAgeChecker(10))

		if len(io.regs) > 1 {
			t.Fatalf("didn't remove expired sessions, got %d/3", len(io.regs))
		}
		if _, ok := io.regs["3"]; !ok {
			t.Errorf("session %v must be in the storage", regs["3"])
		}
	})
}

func createExtSessionFromSession(v *session) *extSession {
	return &extSession{
		v.v,
		v.ct.UnixNano(),
		v.at.UnixNano(),
	}
}

func createSessionsMapAndList(v ...*session) (m map[string]*list.Element, l *list.List) {
	m = map[string]*list.Element{}
	l = list.New()
	for _, s := range v {
		m[s.id] = l.PushBack(&basicSessionInfo{
			s.id,
			s.ct.UnixNano(),
		})
	}
	return
}

type stubMilliAgeChecker int64

func (c stubMilliAgeChecker) ShouldReap(t time.Time) bool {
	return time.Now().UnixMilli()-t.UnixMilli() >= int64(c)
}

func TestDefaultStorageIO(t *testing.T) {
	path := "sessions_from_test"

	io := newStorageIO(path)

	t.Run("creates session file into the file system", func(t *testing.T) {
		sid := "abcde"
		sess, err := io.Create(sid)

		assert.NoError(t, err)
		assert.NotNil(t, sess)

		if sess.id != sid {
			t.Fatalf("didn't get session with id=%s, got id=%s", sid, sess.id)
		}

		file, err := os.Open(filepath.Join(io.path, fmt.Sprintf("%s%s", io.prefix, sid)))
		if err != nil {
			t.Error("cannot open session file (was the file created?)")
		}
		file.Close()
	})
	t.Run("returns session after read from file system", func(t *testing.T) {
		sid := "abcde"
		sess, err := io.Read(sid)

		assert.NoError(t, err)
		assert.NotNil(t, sess)

		if sess.id != sid {
			t.Errorf("didn't get session with id=%s, got id=%s", sid, sess.id)
		}
	})
	t.Run("writes updated session attributes", func(t *testing.T) {
		sess, _ := io.Read("abcde")

		sess.v["role"] = "test"

		err := io.Write(sess)

		assert.NoError(t, err)

		got, _ := io.Read(sess.id)

		if sess.id != got.id || !sess.ct.Equal(got.ct) || !reflect.DeepEqual(sess.v, got.v) {
			t.Errorf("didn't update session, got %s but want %s", writeSessionToString(got), writeSessionToString(sess))
		}
	})
	t.Run("deletes session from the file system", func(t *testing.T) {
		sid := "abcde"
		err := io.Delete(sid)

		assert.NoError(t, err)

		if _, err := os.Open(filepath.Join(io.path, fmt.Sprintf("%s%s", io.prefix, sid))); err == nil {
			t.Error("the session file still exists")
		}
	})
	t.Run("list sessions name asc sorted by creation time", func(t *testing.T) {

		sess1, _ := io.Create("abcde")
		sess2, _ := io.Create("fghij")
		sess3, _ := io.Create("klmno")

		got := io.List()

		assert.NotNil(t, got)

		if len(got) != 3 {
			t.Fatal("expected 3 sessions")
		}

		if !slices.Contains(got, sess1.id) {
			t.Fatalf("expected %v to contains %q", got, sess1.id)
		}

		if !slices.Contains(got, sess2.id) {
			t.Fatalf("expected %v to contains %q", got, sess2.id)
		}

		if !slices.Contains(got, sess3.id) {
			t.Errorf("expected %v to contains %q", got, sess3.id)
		}
	})

	t.Cleanup(func() {
		if err := os.RemoveAll(io.path); err != nil {
			log.Fatalf("cannot clean up after test, %v", err)
		}
	})
}

func writeSessionToString(sess *session) string {
	return fmt.Sprintf("{id=%s, creationtime=%s, values=%+v}", sess.id, sess.ct, sess.v)
}
