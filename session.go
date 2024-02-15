package session

import (
	"errors"
	"time"
)

var (
	ErrNilValueNotAllowed error = errors.New("session: stores nil values into session is not allowed")
)

type DefaultSession struct {
	id string
	ct time.Time
	v  map[string]any
}

func newDefaultSession(id string, ct time.Time, v map[string]any) *DefaultSession {
	return &DefaultSession{id, ct, v}
}

func (s *DefaultSession) Set(key string, value any) error {
	if value == nil {
		return ErrNilValueNotAllowed
	}
	s.v[key] = value
	return nil
}

func (s *DefaultSession) Get(key string) any {
	return s.v[key]
}

func (s *DefaultSession) Delete(key string) error {
	delete(s.v, key)
	return nil
}

func (s *DefaultSession) SessionID() string {
	return s.id
}

func (s *DefaultSession) CreationTime() time.Time {
	return s.ct
}

type defaultSessionBuilder struct{}

func (sb *defaultSessionBuilder) Build(sid string, onSessionUpdate func(sess Session) error) Session {
	return &DefaultSession{
		id: sid,
		v:  make(map[string]any),
	}
}

var DefaultSessionBuilder SessionBuilder = &defaultSessionBuilder{}
