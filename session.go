package session

import "errors"

var (
	ErrNilValueNotAllowed error = errors.New("session: stores nil values into session is not allowed")
)

type Session struct {
	id string
	v  map[string]any
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
	return nil
}

func (s *Session) SessionID() string {
	return s.id
}

type SessionBuilder struct{}

func (sb *SessionBuilder) Build(sid string, onSessionUpdate func(sess ISession) error) ISession {
	return &Session{
		id: sid,
		v:  make(map[string]any),
	}
}
