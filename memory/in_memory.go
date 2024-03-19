package memory

import (
	"container/list"
	"sync"
	"time"

	sessionpkg "github.com/xandalm/go-session"
)

type session struct {
	id string         // session id (sid)
	v  map[string]any // mapped values
	ct time.Time      // creationtime
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
	sessions map[string]*list.Element
	list     *list.List
}

func newStorage() *storage {
	return &storage{
		sessions: map[string]*list.Element{},
		list:     list.New(),
	}
}

// Returns a session or an error if cannot creates a session into the storage.
func (s *storage) CreateSession(sid string) (sessionpkg.Session, error) {
	if sid == "" {
		panic("empty sid")
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

// Returns a session or an error if cannot reads the session from the storage.
func (s *storage) GetSession(sid string) (sessionpkg.Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if elem, ok := s.sessions[sid]; ok {
		sess := elem.Value.(*session)
		return sess, nil
	}
	return nil, nil
}

// Checks if the storage contains the session.
func (s *storage) ContainsSession(sid string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.sessions[sid]
	return ok, nil
}

// Destroys the session from the storage.
func (s *storage) ReapSession(sid string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if elem, ok := s.sessions[sid]; ok {
		delete(s.sessions, sid)
		s.list.Remove(elem)
	}
	return nil
}

// Scans the storage removing expired sessions.
func (s *storage) Deadline(checker sessionpkg.AgeChecker) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for elem := s.list.Back(); elem != nil; elem = s.list.Back() {
		sess := elem.Value.(*session)
		if checker.ShouldReap(sess.ct) {
			delete(s.sessions, sess.id)
			s.list.Remove(elem)
			continue
		}
		break
	}
}

var _storage = newStorage()

// Returns the storage.
func Storage() *storage {
	return _storage
}
