package session

import (
	"reflect"
	"testing"
	"time"

	"github.com/xandalm/go-session/testing/assert"
)

func TestSessionBuilder(t *testing.T) {
	builder := &defaultSessionBuilder{}
	storage := &stubSessionStorage{}
	t.Run("build and return session", func(t *testing.T) {

		sid := "1"
		sess := builder.Build(sid, storage.Save)

		assert.NotNil(t, sess)

		resess, ok := sess.(*defaultSession)

		if !ok {
			t.Fatal("didn't get expected session")
		}

		assert.Equal(t, resess.id, sid)
		assert.NotEmpty(t, resess.ct)
		assert.NotNil(t, resess.v)
	})
	t.Run("expose session", func(t *testing.T) {
		sid := "1"
		creationtime := time.Now()
		sess := &defaultSession{sid, creationtime, map[string]any{}}

		got := builder.Expose(sess)
		want := map[string]any{
			"_session_id":    sid,
			"_creation_time": creationtime,
		}

		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %+v, but want %+v", got, want)
		}
	})
}

func TestSessionID(t *testing.T) {
	t.Run("returns session id", func(t *testing.T) {

		sid := "1"
		sess := &defaultSession{sid, time.Now(), map[string]any{}}

		assert.Equal(t, sess.SessionID(), sid)
	})
}

func TestCreationTime(t *testing.T) {
	t.Run("returns session creation time", func(t *testing.T) {

		ct := time.Now()
		sess := &defaultSession{"1", ct, map[string]any{}}

		assert.Equal(t, sess.CreationTime(), ct)
	})
}

func TestSet(t *testing.T) {
	cases := []struct {
		tname string
		key   string
		value any
		err   error
	}{
		{"set value to the key", "A", "value", nil},
		{"returns error for nil value", "B", nil, ErrNilValueNotAllowed},
	}

	sess := &defaultSession{"1", time.Now(), map[string]any{}}

	for _, c := range cases {
		t.Run(c.tname, func(t *testing.T) {
			err := sess.Set(c.key, c.value)

			if c.err == nil {
				assert.NoError(t, err)
				if sess.v[c.key] != c.value {
					t.Errorf("expected %s to hold %q, but got %q", c.key, c.value, sess.v[c.key])
				}
			} else {
				assert.Error(t, err, c.err)
			}
		})
	}
}

func TestGet(t *testing.T) {
	t.Run("get value from corresponding key", func(t *testing.T) {
		key := "A"
		value := "value"

		sess := &defaultSession{"1", time.Now(), map[string]any{key: value}}

		got := sess.Get(key)

		assert.NotNil(t, got)
		assert.Equal(t, got.(string), value)
	})
}

func TestDelete(t *testing.T) {
	t.Run("remove a pair from session map", func(t *testing.T) {

		sess := &defaultSession{"1", time.Now(), map[string]any{"key": "value"}}

		err := sess.Delete("key")

		assert.NoError(t, err)

		if _, ok := sess.v["key"]; ok {
			t.Errorf("didn't remove pair from session map")
		}
	})
}
