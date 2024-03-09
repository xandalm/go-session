package filesystem

import (
	"fmt"
	"reflect"
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
	}

	err := sess.Delete("key")

	assert.NoError(t, err)

	if _, ok := sess.v["key"]; ok {
		t.Error("didn't delete value")
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
	return nil
}

func (sio *stubStorageIO) Delete(sid string) error {
	delete(sio.regs, sid)
	return nil
}

func TestCreatingSessionInStorage(t *testing.T) {
	t.Run("create session", func(t *testing.T) {
		io := &stubStorageIO{map[string]*extSession{}}
		storage := &storage{
			io,
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

		storage := &storage{
			io: &stubStorageIO{
				map[string]*extSession{
					sid: {
						V:  map[string]any{},
						Ct: time.Now().UnixNano(),
						At: time.Now().UnixNano(),
					},
				},
			},
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

		io := &stubStorageIO{
			map[string]*extSession{
				sid: {
					V:  map[string]any{},
					Ct: time.Now().UnixNano(),
					At: time.Now().UnixNano(),
				},
			},
		}
		storage := &storage{io}

		err := storage.ReapSession(sid)

		assert.NoError(t, err)

		if _, ok := io.regs[sid]; ok {
			t.Error("didn't remove session")
		}
	})
}
