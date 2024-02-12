package session_test

import (
	"errors"
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

func (ss *StubSessionStorage) Get(sid string) (session.ISession, error) {
	sess := ss.sessions[sid]
	return sess, nil
}

type StubFailingSessionStorage struct{}

var errFoo error = errors.New("foo error")

func (ss *StubFailingSessionStorage) Save(sess session.ISession) error {
	return errFoo
}

func (ss *StubFailingSessionStorage) Get(sid string) (session.ISession, error) {
	return nil, errFoo
}

func TestSessionInit(t *testing.T) {

	sessionBuilder := &StubSessionBuilder{}
	sessionStorage := &StubSessionStorage{}

	provider := session.NewProvider(sessionBuilder, sessionStorage)

	t.Run("init, store and returns session", func(t *testing.T) {

		sid := "17af454"
		session, err := provider.SessionInit(sid)

		assertNoError(t, err)
		assertNotNil(t, session)

		if _, ok := sessionStorage.sessions[sid]; !ok {
			t.Error("didn't stores the session")
		}
	})
	t.Run("returns error for empty sid", func(t *testing.T) {

		_, err := provider.SessionInit("")

		assertError(t, err, session.ErrEmptySessionId)
	})
}

func TestSessionRead(t *testing.T) {

	sessionBuilder := &StubSessionBuilder{}
	sessionStorage := &StubSessionStorage{
		sessions: map[string]session.ISession{
			"17af454": &StubSession{
				id: "17af454",
			},
		},
	}

	provider := session.NewProvider(sessionBuilder, sessionStorage)

	t.Run("returns stored session", func(t *testing.T) {
		sid := "17af454"
		session, err := provider.SessionRead(sid)

		assertNoError(t, err)
		assertNotNil(t, session)

		if session.SessionID() != sid {
			t.Errorf("didn't get expected session, got %s but want %s", session.SessionID(), sid)
		}
	})
	t.Run("returns error on failing session restoration", func(t *testing.T) {
		provider := session.NewProvider(&StubSessionBuilder{}, &StubFailingSessionStorage{})

		_, err := provider.SessionRead("17af454")

		assertError(t, err, session.ErrRestoringSession)
	})
}

func assertNoError(t testing.TB, err error) {
	t.Helper()

	if err != nil {
		t.Fatalf("expect no error, got %v", err)
	}
}

func assertError(t testing.TB, got, want error) {
	t.Helper()

	if got == nil {
		t.Fatalf("expect error, but didn't got one")
	}

	if got != want {
		t.Fatalf("got error %v but want %v", got, want)
	}
}
