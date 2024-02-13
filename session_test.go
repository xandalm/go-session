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

		assert.AssertNotNil(t, sess)
	})
}

func TestSessionID(t *testing.T) {
	builder := &SessionBuilder{}
	storage := &stubSessionStorage{}
	t.Run("returns session id", func(t *testing.T) {
		sid := "1"
		got := builder.Build(sid, storage.Save)

		assert.AssertNotNil(t, got)

		assert.AssertEqual(t, got.SessionID(), sid)
	})
}

func TestSet(t *testing.T) {
	builder := &SessionBuilder{}
	storage := &stubSessionStorage{}
	sess := builder.Build("1", storage.Save)
	assert.AssertNotNil(t, sess)
	t.Run("sets session value", func(t *testing.T) {
		key := "A"
		value := "value"
		err := sess.Set(key, value)

		assert.AssertNoError(t, err)

		resess := sess.(*Session)
		if resess.v[key] != value {
			t.Errorf("expected %s to hold %q, but got %q", key, value, resess.v[key])
		}
	})
}
