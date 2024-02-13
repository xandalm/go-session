package session

import (
	"testing"

	"github.com/xandalm/go-session/testing/assert"
)

func TestSessionBuilder(t *testing.T) {
	builder := &SessionBuilder{}
	storage := &stubSessionStorage{}
	t.Run("build and return session", func(t *testing.T) {

		sid := "1"
		sess := builder.Build(sid, storage.Save)

		assert.NotNil(t, sess)
	})
}

func TestSessionID(t *testing.T) {
	builder := &SessionBuilder{}
	storage := &stubSessionStorage{}
	t.Run("returns session id", func(t *testing.T) {
		sid := "1"
		got := builder.Build(sid, storage.Save)

		assert.NotNil(t, got)

		assert.Equal(t, got.SessionID(), sid)
	})
}

func TestSet(t *testing.T) {
	t.Run("set value to the key", func(t *testing.T) {
		sess := &Session{
			id: "1",
			v:  map[string]any{},
		}

		key := "A"
		value := "value"
		err := sess.Set(key, value)

		assert.NoError(t, err)

		if sess.v[key] != value {
			t.Errorf("expected %s to hold %q, but got %q", key, value, sess.v[key])
		}
	})
}

func TestGet(t *testing.T) {
	t.Run("get value from corresponding key", func(t *testing.T) {
		key := "A"
		value := "value"

		sess := &Session{
			id: "1",
			v: map[string]any{
				key: value,
			},
		}

		got := sess.Get(key)

		assert.NotNil(t, got)
		assert.Equal(t, got.(string), value)
	})
}
