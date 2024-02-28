package memory

import (
	"container/list"
	"sync"
	"time"

	sessionpkg "github.com/xandalm/go-session"
)

type session struct {
	id string
	v  map[string]any
	ta time.Time
}

func newSession(sid string) *session {
	return &session{
		id: sid,
		v:  map[string]any{},
		ta: time.Now(),
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
	sessions map[string]*list.Element
	list     *list.List
}

func newStorage() *storage {
	return &storage{
		sessions: map[string]*list.Element{},
		list:     list.New(),
	}
}

func (s *storage) CreateSession(sid string) (sessionpkg.Session, error) {
	if sid == "" {
		panic("session: empty sid (session id)")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	sess := newSession(sid)
	if err := s.insertSession(sess); err != nil {
		return nil, err
	}
	return sess, nil
}

func (s *storage) insertSession(sess *session) error {
	elem := s.list.PushFront(sess)
	s.sessions[sess.id] = elem
	return nil
}

func (s *storage) GetSession(sid string) (sessionpkg.Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if elem, ok := s.sessions[sid]; ok {
		sess := elem.Value.(*session)
		return sess, nil
	}
	return nil, nil
}

func (s *storage) ReapSession(sid string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if elem, ok := s.sessions[sid]; ok {
		delete(s.sessions, sid)
		s.list.Remove(elem)
	}
	return nil
}

func (s *storage) Deadline(checker sessionpkg.AgeChecker) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for elem := s.list.Back(); elem != nil; elem = s.list.Back() {
		sess := elem.Value.(*session)
		if checker.ShouldReap(sess.ta) {
			delete(s.sessions, sess.id)
			s.list.Remove(elem)
			continue
		}
		break
	}
}

var Storage = newStorage()
