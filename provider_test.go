package session

import (
	"testing"
	"time"

	"github.com/xandalm/go-session/testing/assert"
)

func TestSessionInit(t *testing.T) {

	sessionBuilder := &stubSessionBuilder{}
	sessionStorage := &stubSessionStorage{}

	provider := NewProvider(sessionBuilder, sessionStorage)

	t.Run("init, store and returns session", func(t *testing.T) {

		sid := "17af454"
		session, err := provider.SessionInit(sid)

		assert.AssertNoError(t, err)
		assert.AssertNotNil(t, session)

		if _, ok := sessionStorage.Sessions[sid]; !ok {
			t.Error("didn't stores the session")
		}
	})
	t.Run("returns error for empty sid", func(t *testing.T) {

		_, err := provider.SessionInit("")

		assert.AssertError(t, err, ErrEmptySessionId)
	})
	t.Run("returns error for duplicated sid", func(t *testing.T) {
		_, err := provider.SessionInit("17af454")

		assert.AssertError(t, err, ErrDuplicateSessionId)
	})
	t.Run("returns error for inability to ensure non-duplicity", func(t *testing.T) {
		sessionStorage := &stubFailingSessionStorage{
			Sessions: map[string]ISession{
				"17af454": &stubSession{
					Id: "17af454",
				},
			},
		}
		provider := NewProvider(sessionBuilder, sessionStorage)

		_, err := provider.SessionInit("17af454")

		assert.AssertError(t, err, ErrUnableToEnsureNonDuplicity)
	})
	t.Run("returns error for storage save failure", func(t *testing.T) {
		sessionStorage := &mockSessionStorage{
			Sessions: map[string]ISession{},
			GetFunc:  func(sid string) (ISession, error) { return nil, nil },
			SaveFunc: func(sess ISession) error { return ErrFoo },
		}
		provider := NewProvider(sessionBuilder, sessionStorage)

		_, err := provider.SessionInit("17af450")

		assert.AssertError(t, err, ErrUnableToSaveSession)
	})
}

func TestSessionRead(t *testing.T) {

	sessionBuilder := &stubSessionBuilder{}
	sessionStorage := &stubSessionStorage{
		Sessions: map[string]ISession{
			"17af454": &stubSession{
				Id: "17af454",
			},
		},
	}

	provider := NewProvider(sessionBuilder, sessionStorage)

	t.Run("returns stored session", func(t *testing.T) {
		sid := "17af454"
		session, err := provider.SessionRead(sid)

		assert.AssertNoError(t, err)
		assert.AssertNotNil(t, session)

		if session.SessionID() != sid {
			t.Errorf("didn't get expected session, got %s but want %s", session.SessionID(), sid)
		}
	})
	t.Run("returns error on failing session restoration", func(t *testing.T) {
		provider := NewProvider(&stubSessionBuilder{}, &stubFailingSessionStorage{})

		_, err := provider.SessionRead("17af454")

		assert.AssertError(t, err, ErrRestoringSession)
	})
}

func TestSessionDestroy(t *testing.T) {

	sessionBuilder := &stubSessionBuilder{}
	sessionStorage := &stubSessionStorage{
		Sessions: map[string]ISession{
			"17af454": &stubSession{
				Id: "17af454",
			},
		},
	}

	provider := NewProvider(sessionBuilder, sessionStorage)

	t.Run("destroys session", func(t *testing.T) {
		sid := "17af454"
		err := provider.SessionDestroy(sid)

		assert.AssertNoError(t, err)

		if _, ok := sessionStorage.Sessions[sid]; ok {
			t.Fatalf("didn't destroy session")
		}
	})
	t.Run("returns error for destroy failing", func(t *testing.T) {
		provider := NewProvider(&stubSessionBuilder{}, &stubFailingSessionStorage{})

		err := provider.SessionDestroy("17af454")

		assert.AssertError(t, err, ErrUnableToDestroySession)
	})
}

func TestSessionGC(t *testing.T) {

	sess := &stubSession{
		Id:        "17af454",
		CreatedAt: time.Now(),
	}

	sessionBuilder := &stubSessionBuilder{}
	sessionStorage := &stubSessionStorage{
		Sessions: map[string]ISession{
			"17af454": sess,
		},
	}

	provider := NewProvider(sessionBuilder, sessionStorage)

	t.Run("destroy sessions that arrives max age", func(t *testing.T) {

		time.Sleep(1 * time.Second)

		provider.SessionGC(1)

		if _, ok := sessionStorage.Sessions[sess.Id]; ok {
			t.Error("didn't destroy session")
		}
	})
}
