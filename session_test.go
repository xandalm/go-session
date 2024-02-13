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
