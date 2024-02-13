package session_test

import (
	"testing"

	"github.com/xandalm/go-session"
)

func TestSessionBuilder(t *testing.T) {
	builder := &session.SessionBuilder{}
	t.Run("build and return session", func(t *testing.T) {

		sid := "1"
		sess := builder.Build(sid)

		assertNotNil(t, sess)
	})
}
