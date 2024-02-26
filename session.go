package session

import (
	"errors"
	"maps"
	"time"
)

var (
	ErrNilValueNotAllowed error = errors.New("session: stores nil values into session is not allowed")
)

type SessionStorageUpdater func(Session) error

type defaultSession struct {
	id string
	ct time.Time
	v  SessionValues
	fn SessionStorageUpdater
}

func newDefaultSession(id string, storage Storage) *defaultSession {
	return &defaultSession{id, time.Now(), make(SessionValues), storage.Save}
}

func (s *defaultSession) Set(key string, value any) error {
	if value == nil {
		return ErrNilValueNotAllowed
	}
	s.v[key] = value
	s.fn(s)
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

func (sb *defaultSessionBuilder) Build(sid string, storage Storage) Session {
	return newDefaultSession(sid, storage)
}

func (sb *defaultSessionBuilder) Restore(sid string, creationTime time.Time, values SessionValues, onSessionUpdate func(sess Session) error) (Session, error) {
	return &defaultSession{
		sid,
		creationTime,
		values,
		nil,
	}, nil
}

var DefaultSessionBuilder SessionBuilder = &defaultSessionBuilder{}
