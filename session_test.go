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
	cases := []struct {
		tname string
		key   string
		value any
		err   error
	}{
		{"set value to the key", "A", "value", nil},
		{"returns error for nil value", "B", nil, ErrNilValueNotAllowed},
	}

	sess := &Session{
		id: "1",
		v:  map[string]any{},
	}

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
