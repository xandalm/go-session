package session

import "sync"

type MemoryStorage struct {
	mu       sync.Mutex
	sessions map[string]ISession
}

func (s *MemoryStorage) Save(sess ISession) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[sess.SessionID()] = sess
	return nil
}

func (s *MemoryStorage) Get(sid string) (ISession, error) {
	return s.sessions[sid], nil
}

func (s *MemoryStorage) Rip(sid string) error {
	delete(s.sessions, sid)
	return nil
}
