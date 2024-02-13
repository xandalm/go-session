package session_test

import (
	"testing"

	"github.com/xandalm/go-session"
)

func TestSessionBuilder(t *testing.T) {
	builder := &session.SessionBuilder{}
	storage := &StubSessionStorage{}
	t.Run("build and return session", func(t *testing.T) {

		sid := "1"
		sess := builder.Build(sid, storage.Save)

		assertNotNil(t, sess)
	})

	t.Run("returns provider", func(t *testing.T) {
		provider := session.NewProvider(builder, storage)

		assertNotNil(t, provider)
	})
}

func TestSessionID(t *testing.T) {
	builder := &session.SessionBuilder{}
	storage := &StubSessionStorage{}
	t.Run("returns session id", func(t *testing.T) {
		sid := "1"
		got := builder.Build(sid, storage.Save)

		assertNotNil(t, got)

		assertEqual(t, got.SessionID(), sid)
	})
}
