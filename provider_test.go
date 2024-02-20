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
	t.Run("call build and save supporters", func(t *testing.T) {
		sessionBuilder := &spySessionBuilder{}
		sessionStorage := &spySessionStorage{}

		provider := &DefaultProvider{sessionBuilder, sessionStorage, dummyAdapter}

		_, err := provider.SessionInit("1")
		assert.NoError(t, err)

		if sessionBuilder.callsToBuild == 0 {
			t.Fatal("didn't call builder to build")
		}
		if sessionStorage.callsToSave == 0 {
			t.Error("didn't call storage to save")
		}
	})

	sessionBuilder := &stubSessionBuilder{}
	sessionStorage := &stubSessionStorage{}

	provider := &DefaultProvider{sessionBuilder, sessionStorage, dummyAdapter}
	t.Run("init, store and returns session", func(t *testing.T) {

		sid := "17af454"
		session, err := provider.SessionInit(sid)

		assert.NoError(t, err)
		assert.NotNil(t, session)

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

		assert.Error(t, err, ErrDuplicateSessionId)
	})
	t.Run("returns error for inability to ensure non-duplicity", func(t *testing.T) {
		sessionStorage := &stubFailingSessionStorage{
			Sessions: map[string]Session{
				"17af454": &stubSession{
					Id: "17af454",
				},
			},
		}
		provider := &DefaultProvider{sessionBuilder, sessionStorage, dummyAdapter}

		_, err := provider.SessionInit("17af454")

		assert.Error(t, err, ErrUnableToEnsureNonDuplicity)
	})
	t.Run("returns error for storage save failure", func(t *testing.T) {
		sessionStorage := &mockSessionStorage{
			Sessions: map[string]Session{},
			GetFunc:  func(sid string) (Session, error) { return nil, nil },
			SaveFunc: func(sess Session) error { return errFoo },
		}
		provider := &DefaultProvider{sessionBuilder, sessionStorage, dummyAdapter}

		_, err := provider.SessionInit("17af450")

		assert.Error(t, err, ErrUnableToSaveSession)
	})
}

func TestSessionRead(t *testing.T) {

	sessionBuilder := &stubSessionBuilder{}
	sessionStorage := &stubSessionStorage{
		Sessions: map[string]Session{
			"17af454": &stubSession{
				Id: "17af454",
			},
		},
	}

	provider := &DefaultProvider{sessionBuilder, sessionStorage, dummyAdapter}

	t.Run("returns stored session", func(t *testing.T) {
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
		sessionBuilder := &stubSessionBuilder{}
		sessionStorage := &stubFailingSessionStorage{}
		provider := &DefaultProvider{sessionBuilder, sessionStorage, dummyAdapter}

		_, err := provider.SessionRead("17af454")

		assert.Error(t, err, ErrUnableToRestoreSession)
	})
}

func TestSessionDestroy(t *testing.T) {

	sessionBuilder := &stubSessionBuilder{}
	sessionStorage := &stubSessionStorage{
		Sessions: map[string]Session{
			"17af454": &stubSession{
				Id: "17af454",
			},
		},
	}

	provider := &DefaultProvider{sessionBuilder, sessionStorage, dummyAdapter}

	t.Run("destroys session", func(t *testing.T) {
		sid := "17af454"
		err := provider.SessionDestroy(sid)

		assert.NoError(t, err)

		if _, ok := sessionStorage.Sessions[sid]; ok {
			t.Fatalf("didn't destroy session")
		}
	})
	t.Run("returns error for destroy failing", func(t *testing.T) {
		sessionBuilder := &stubSessionBuilder{}
		sessionStorage := &stubFailingSessionStorage{}
		provider := &DefaultProvider{sessionBuilder, sessionStorage, dummyAdapter}

		err := provider.SessionDestroy("17af454")

		assert.Error(t, err, ErrUnableToDestroySession)
	})
}

func TestSessionGC(t *testing.T) {

	t.Run("destroy sessions that arrives max age", func(t *testing.T) {

		sid1 := "17af450"
		sid2 := "17af454"

		sessionBuilder := &stubSessionBuilder{}
		sessionStorage := &stubSessionStorage{
			Sessions: map[string]Session{},
		}

		provider := &DefaultProvider{sessionBuilder, sessionStorage, func(maxAge int64) AgeChecker {
			return stubMilliAgeChecker(maxAge)
		}}

		sessionStorage.Sessions[sid1] = newStubSession(sid1, time.Now(), nil)

		time.Sleep(2 * time.Millisecond)

		sessionStorage.Sessions[sid2] = newStubSession(sid2, time.Now(), nil)

		provider.SessionGC(1)

		if _, ok := sessionStorage.Sessions[sid1]; ok {
			t.Fatal("didn't destroy session")
		}

		if len(sessionStorage.Sessions) != 1 {
			t.Errorf("expected the session with id=%s in storage", sid2)
		}
	})
}
