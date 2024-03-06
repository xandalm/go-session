package filesystem

import (
	"bytes"
	"encoding/gob"
	"log"
	"os"
	"path/filepath"
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
	err := sess.Set("foo", "bar")

	assert.NoError(t, err)
	want := "bar"

	if got, ok := sess.v["foo"]; ok {
		if got != want {
			t.Errorf("got value %q, but want %q", got, want)
		}
		return
	}
	t.Error("didn't set value")
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

func TestCreatingSessionInStorage(t *testing.T) {
	path := ""
	dir := "sessions"
	ext := "sess"
	t.Run("create session", func(t *testing.T) {
		storage := NewStorage(path, dir, ext)

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
		if _, err := os.ReadFile(joinPath(path, dir, sid+".sess")); err != nil {
			t.Error("cannot open session file")
		}
	})
	t.Cleanup(func() {
		if err := os.RemoveAll(joinPath(path, dir)); err != nil {
			log.Fatalf("didn't complete clean up, %v", err)
		}
	})
}

func TestReadSession(t *testing.T) {
	dummyPath := ""
	dummyDir := "sessions"
	dummyExt := "sess"

	now := time.Now()
	sess := &session{
		id: "abcde",
		v:  map[string]any{},
		ct: now,
		at: now,
	}

	var buf bytes.Buffer

	enc := gob.NewEncoder(&buf)
	err := enc.Encode(sess)
	if err != nil {
		log.Fatalf("expected no error, %v", err)
	}

	storage := NewStorage(dummyPath, dummyDir, dummyExt)

	got, err := storage.readSession(&buf)

	assert.NoError(t, err)
	assert.NotEmpty(t, got.ct)
	assert.NotEmpty(t, got.at)

	if !sess.ct.Equal(got.ct) {
		t.Fatalf("got creation time %s, but want %s", got.ct, sess.ct)
	}

	if !sess.at.Equal(got.at) {
		t.Errorf("got access time %s, but want %s", got.at, sess.at)
	}
}

func TestGettingSessionFromStorage(t *testing.T) {
	path := ""
	dir := "sessions"
	ext := "sess"

	sid := "abcde"

	if err := makeDir(path, dir); err != nil {
		log.Fatalf("cannot create storage folder, %v", err)
	}

	if err := makeSessionFile(joinPath(path, dir), sid, ext); err != nil {
		log.Fatalf("cannot create session file, %v", err)
	}

	t.Run("returns session", func(t *testing.T) {
		storage := NewStorage(path, dir, ext)

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

	t.Cleanup(func() {
		if err := os.RemoveAll(joinPath(path, dir)); err != nil {
			log.Fatalf("didn't complete clean up, %v", err)
		}
	})
}

func joinPath(v ...string) string {
	return filepath.Join(v...)
}

func makeDir(path, dir string) error {
	if err := os.MkdirAll(joinPath(path, dir), 0750); err != nil {
		return err
	}
	return nil
}

func makeSessionFile(path, sid, ext string) error {
	f, err := os.OpenFile(joinPath(path, sid+"."+ext), os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer f.Close()
	return nil
}
