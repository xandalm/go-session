package memory

import (
	"sync"
	"time"

	sessionpkg "github.com/xandalm/go-session"
)

type session struct {
	id string
	v  map[string]any
	ct time.Time
}

func newSession(sid string) *session {
	return &session{
		id: sid,
		v:  map[string]any{},
		ct: time.Now(),
	}
}

func (s *session) SessionID() string {
	return s.id
}

func (s *session) Get(key string) any {
	return s.v[key]
}

func (s *session) Set(key string, value any) error {
	s.v[key] = value
	return nil
}

func (s *session) Delete(key string) error {
	delete(s.v, key)
	return nil
}

type storage struct {
	mu       sync.Mutex
	sessions map[string]*session
}

func (s *storage) CreateSession(sid string) (sessionpkg.Session, error) {
	if sid == "" {
		panic("session: empty sid (session id)")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	sess := newSession(sid)
	s.sessions[sess.id] = sess
	return sess, nil
}

func (s *storage) GetSession(sid string) (sessionpkg.Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sess, ok := s.sessions[sid]; ok {
		return sess, nil
	}
	return nil, nil
}

func (s *storage) ReapSession(sid string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, sid)
	return nil
}

func (s *storage) Deadline(checker sessionpkg.AgeChecker) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for sid, sess := range s.sessions {
		if checker.ShouldReap(sess.ct) {
			delete(s.sessions, sid)
		}
	}
}

var Storage = &storage{
	sessions: map[string]*session{},
}
