package session

import (
	"errors"
	"maps"
	"time"
)

var (
	ErrNilValueNotAllowed error = errors.New("session: stores nil values into session is not allowed")
)

type defaultSession struct {
	id string
	ct time.Time
	v  SessionValues
}

func newDefaultSession(id string) *defaultSession {
	return &defaultSession{id, time.Now(), make(SessionValues)}
}

func (s *defaultSession) Set(key string, value any) error {
	if value == nil {
		return ErrNilValueNotAllowed
	}
	s.v[key] = value
	return nil
}

func (s *defaultSession) Get(key string) any {
	return s.v[key]
}

func (s *defaultSession) Delete(key string) error {
	delete(s.v, key)
	return nil
}

func (s *defaultSession) Values() SessionValues {
	return maps.Clone(s.v)
}

func (s *defaultSession) SessionID() string {
	return s.id
}

func (s *defaultSession) CreationTime() time.Time {
	return s.ct
}

type defaultSessionBuilder struct{}

func (sb *defaultSessionBuilder) Build(sid string, onSessionUpdate func(sess Session) error) Session {
	return newDefaultSession(sid)
}

func (sb *defaultSessionBuilder) Restore(sid string, creationTime time.Time, values SessionValues, onSessionUpdate func(sess Session) error) (Session, error) {
	return &defaultSession{
		sid,
		creationTime,
		values,
	}, nil
}

var DefaultSessionBuilder SessionBuilder = &defaultSessionBuilder{}
