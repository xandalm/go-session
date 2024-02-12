package session_test

import (
	"testing"

	"github.com/xandalm/go-session"
)

type StubSessionBuilder struct {
}

func (sb *StubSessionBuilder) Build(sid string) session.ISession {
	return &StubSession{
		id: sid,
	}
}

type StubSessionStorage struct {
	sessions map[string]session.ISession
}

func (ss *StubSessionStorage) Save(sess session.ISession) error {
	if ss.sessions == nil {
		ss.sessions = make(map[string]session.ISession)
	}
	ss.sessions[sess.SessionID()] = sess
	return nil
}

func TestSessionInit(t *testing.T) {
	t.Run("returns session", func(t *testing.T) {
		sessionBuilder := &StubSessionBuilder{}
		sessionStorage := &StubSessionStorage{}

		provider := session.NewProvider(sessionBuilder, sessionStorage)

		sid := "17af454"
		session, _ := provider.SessionInit(sid)

		assertNotNil(t, session)

		if _, ok := sessionStorage.sessions[sid]; !ok {
			t.Error("didn't stores the session")
		}
	})
}
