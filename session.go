package session

import (
	"errors"
	"time"
)

var (
	ErrNilValueNotAllowed error = errors.New("session: stores nil values into session is not allowed")
)

type Session struct {
	id string
	ct time.Time
	v  map[string]any
}

func newSession(id string, ct time.Time, v map[string]any) *Session {
	return &Session{id, ct, v}
}

func (s *Session) Set(key string, value any) error {
	if value == nil {
		return ErrNilValueNotAllowed
	}
	s.v[key] = value
	return nil
}

func (s *Session) Get(key string) any {
	return s.v[key]
}

func (s *Session) Delete(key string) error {
	delete(s.v, key)
	return nil
}

func (s *Session) SessionID() string {
	return s.id
}

func (s *Session) CreationTime() time.Time {
	return s.ct
}

type SessionBuilder struct{}

func (sb *SessionBuilder) Build(sid string, onSessionUpdate func(sess ISession) error) ISession {
	return &Session{
		id: sid,
		v:  make(map[string]any),
	}
}
