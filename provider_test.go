package session

import (
	"testing"
	"time"

	"github.com/xandalm/go-session/testing/assert"
)

var dummyAdapter = func(maxAge int64) AgeChecker {
	return nil
}

func TestSessionInit(t *testing.T) {
	t.Run("tell storage to create session", func(t *testing.T) {
		sessionStorage := &spySessionStorage{}

		provider := &defaultProvider{sessionStorage, dummyAdapter}

		_, err := provider.SessionInit("1")
		assert.NoError(t, err)

		if sessionStorage.callsToCreateSession == 0 {
			t.Error("didn't tell storage")
		}
	})

	sessionStorage := newStubSessionStorage()

	provider := &defaultProvider{sessionStorage, dummyAdapter}
	t.Run("init the session", func(t *testing.T) {

		sid := "17af454"
		sess, err := provider.SessionInit(sid)

		assert.NoError(t, err)
		assert.NotNil(t, sess)

		if _, ok := sessionStorage.Sessions[sid]; !ok {
			t.Error("didn't stores the session")
		}
	})
	t.Run("returns error for empty sid", func(t *testing.T) {

		_, err := provider.SessionInit("")

		assert.Error(t, err, ErrEmptySessionId)
	})
	t.Run("returns error for duplicated sid", func(t *testing.T) {
		_, err := provider.SessionInit("17af454")

		assert.Error(t, err, ErrDuplicatedSessionId)
	})
	t.Run("returns error for inability to ensure non-duplicity", func(t *testing.T) {
		sessionStorage := &stubFailingSessionStorage{
			Sessions: map[string]Session{
				"17af454": &stubSession{
					Id: "17af454",
				},
			},
		}
		provider := &defaultProvider{sessionStorage, dummyAdapter}

		_, err := provider.SessionInit("17af454")

		assert.Error(t, err, ErrUnableToEnsureNonDuplicity)
	})
	t.Run("returns error for storage create failure", func(t *testing.T) {
		sessionStorage := &mockSessionStorage{
			Sessions:            map[string]Session{},
			CreateSessionFunc:   func(sid string) (Session, error) { return nil, errFoo },
			GetSessionFunc:      func(sid string) (Session, error) { return nil, nil },
			ContainsSessionFunc: func(sid string) (bool, error) { return false, nil },
		}
		provider := &defaultProvider{sessionStorage, dummyAdapter}

		_, err := provider.SessionInit("17af450")

		assert.Error(t, err, ErrUnableToSaveSession)
	})
}

func TestSessionRead(t *testing.T) {

	t.Run("tell storage to get session", func(t *testing.T) {
		sessionStorage := &spySessionStorage{}

		provider := &defaultProvider{sessionStorage, dummyAdapter}

		_, err := provider.SessionRead("1")
		assert.NoError(t, err)

		if sessionStorage.callsToGetSession == 0 {
			t.Error("didn't tell storage")
		}
	})

	sessionStorage := &stubSessionStorage{
		Sessions: map[string]*stubSession{
			"17af454": {
				Id: "17af454",
			},
		},
	}

	provider := &defaultProvider{sessionStorage, dummyAdapter}

	t.Run("returns session", func(t *testing.T) {
		sid := "17af454"
		session, err := provider.SessionRead(sid)

		assert.NoError(t, err)
		assert.NotNil(t, session)

		if session.SessionID() != sid {
			t.Errorf("didn't get expected session, got %s but want %s", session.SessionID(), sid)
		}
	})
	t.Run("start new session if has no session to read", func(t *testing.T) {
		sid := "17af450"
		session, err := provider.SessionRead(sid)

		assert.NoError(t, err)
		assert.NotNil(t, session)

		if session.SessionID() != sid {
			t.Errorf("didn't get expected session, got %s but want %s", session.SessionID(), sid)
		}
	})
	t.Run("returns error on failing session restoration", func(t *testing.T) {
		sessionStorage := &stubFailingSessionStorage{}
		provider := &defaultProvider{sessionStorage, dummyAdapter}

		_, err := provider.SessionRead("17af454")

		assert.Error(t, err, ErrUnableToRestoreSession)
	})
}

func TestSessionDestroy(t *testing.T) {

	sessionStorage := &stubSessionStorage{
		Sessions: map[string]*stubSession{
			"17af454": {
				Id: "17af454",
			},
		},
	}

	provider := &defaultProvider{sessionStorage, dummyAdapter}

	t.Run("destroys session", func(t *testing.T) {
		sid := "17af454"
		err := provider.SessionDestroy(sid)

		assert.NoError(t, err)

		if _, ok := sessionStorage.Sessions[sid]; ok {
			t.Fatalf("didn't destroy session")
		}
	})
	t.Run("returns error for destroy failing", func(t *testing.T) {
		sessionStorage := &stubFailingSessionStorage{}
		provider := &defaultProvider{sessionStorage, dummyAdapter}

		err := provider.SessionDestroy("17af454")

		assert.Error(t, err, ErrUnableToDestroySession)
	})
}

func TestSessionGC(t *testing.T) {

	t.Run("destroy sessions that arrives max age", func(t *testing.T) {

		sid1 := "17af450"
		sid2 := "17af454"

		sessionStorage := &stubSessionStorage{
			Sessions: map[string]*stubSession{},
		}

		provider := &defaultProvider{sessionStorage, func(maxAge int64) AgeChecker {
			return stubMilliAgeChecker(maxAge)
		}}

		sessionStorage.Sessions[sid1] = newStubSession(sid1)

		time.Sleep(2 * time.Millisecond)

		sessionStorage.Sessions[sid2] = newStubSession(sid2)

		provider.SessionGC(1)

		if _, ok := sessionStorage.Sessions[sid1]; ok {
			t.Fatal("didn't destroy session")
		}

		if len(sessionStorage.Sessions) != 1 {
			t.Errorf("expected the session with id=%s in storage", sid2)
		}
	})
}
