package session

import (
	"errors"
	"time"
)

var (
	ErrNilValueNotAllowed error = errors.New("session: stores nil values into session is not allowed")
)

type defaultSession struct {
	id string
	ct time.Time
	v  map[string]any
}

func newDefaultSession(id string) *defaultSession {
	return &defaultSession{id, time.Now(), make(map[string]any)}
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

func (sb *defaultSessionBuilder) Expose(sess Session) map[string]any {
	res := make(map[string]any)

	_sess, ok := sess.(*defaultSession)
	if !ok {
		panic("session: cannot expose session because incompatibility")
	}

	res["_session_id"] = _sess.SessionID()
	res["_creation_time"] = _sess.CreationTime()
	for key, value := range _sess.v {
		res[key] = value
	}

	return res
}

var DefaultSessionBuilder SessionBuilder = &defaultSessionBuilder{}
