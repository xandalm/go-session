package session_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/xandalm/go-session"
)

type StubSessionBuilder struct {
}

func (sb *StubSessionBuilder) Build(sid string, onUpdate func(session.ISession) error) session.ISession {
	return &StubSession{
		id:       sid,
		onUpdate: onUpdate,
	}
}

type StubSessionStorage struct {
	sessions  map[string]session.ISession
	destroyed []session.ISession
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

func (ss *StubSessionStorage) Destroy(sid string) error {
	sess, ok := ss.sessions[sid]
	if !ok {
		return fmt.Errorf("unknown sid")
	}
	ss.destroyed = append(ss.destroyed, sess)
	return nil
}

type StubFailingSessionStorage struct {
	sessions map[string]session.ISession
}

var errFoo error = errors.New("foo error")

func (ss *StubFailingSessionStorage) Save(sess session.ISession) error {
	return errFoo
}

func (ss *StubFailingSessionStorage) Get(sid string) (session.ISession, error) {
	return nil, errFoo
}

func (ss *StubFailingSessionStorage) Destroy(sid string) error {
	return errFoo
}

type MockSessionStorage struct {
	sessions    map[string]session.ISession
	SaveFunc    func(sess session.ISession) error
	GetFunc     func(sid string) (session.ISession, error)
	DestroyFunc func(sid string) error
}

func (ss *MockSessionStorage) Save(sess session.ISession) error {
	return ss.SaveFunc(sess)
}

func (ss *MockSessionStorage) Get(sid string) (session.ISession, error) {
	return ss.GetFunc(sid)
}

func (ss *MockSessionStorage) Destroy(sid string) error {
	return ss.DestroyFunc(sid)
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
	t.Run("returns error for duplicated sid", func(t *testing.T) {
		_, err := provider.SessionInit("17af454")

		assertError(t, err, session.ErrDuplicateSessionId)
	})
	t.Run("returns error for inability to ensure non-duplicity", func(t *testing.T) {
		sessionStorage := &StubFailingSessionStorage{
			sessions: map[string]session.ISession{
				"17af454": &StubSession{
					id: "17af454",
				},
			},
		}
		provider := session.NewProvider(sessionBuilder, sessionStorage)

		_, err := provider.SessionInit("17af454")

		assertError(t, err, session.ErrUnableToEnsureNonDuplicity)
	})
	t.Run("returns error for storage save failure", func(t *testing.T) {
		sessionStorage := &MockSessionStorage{
			sessions: map[string]session.ISession{},
			GetFunc:  func(sid string) (session.ISession, error) { return nil, nil },
			SaveFunc: func(sess session.ISession) error { return errFoo },
		}
		provider := session.NewProvider(sessionBuilder, sessionStorage)

		_, err := provider.SessionInit("17af450")

		assertError(t, err, session.ErrUnableToSaveSession)
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

func TestSessionDestroy(t *testing.T) {

	sessionBuilder := &StubSessionBuilder{}
	sessionStorage := &StubSessionStorage{
		sessions: map[string]session.ISession{
			"17af454": &StubSession{
				id: "17af454",
			},
		},
	}

	provider := session.NewProvider(sessionBuilder, sessionStorage)

	t.Run("set session to be destroyed", func(t *testing.T) {
		sid := "17af454"
		err := provider.SessionDestroy(sid)

		assertNoError(t, err)

		sess := sessionStorage.sessions[sid]
		assertContains(t, sessionStorage.destroyed, sess, func(a, b session.ISession) bool {
			return a.SessionID() == b.SessionID()
		})
	})
	t.Run("returns error for destroy failing", func(t *testing.T) {
		provider := session.NewProvider(&StubSessionBuilder{}, &StubFailingSessionStorage{})

		err := provider.SessionDestroy("17af454")

		assertError(t, err, session.ErrUnableToDestroySession)
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
		t.Fatalf(`got error "%v" but want "%v"`, got, want)
	}
}

func assertContains[T any](t testing.TB, haystack []T, v T, predicate func(a, b T) bool) {
	t.Helper()

	for _, o := range haystack {
		if predicate(o, v) {
			return
		}
	}

	t.Fatalf("didn't contains %v in %+v", v, haystack)
}
