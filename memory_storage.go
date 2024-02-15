package session

import (
	"sync"
)

type MemoryStorage struct {
	mu       sync.Mutex
	sessions map[string]Session
	list     []Session
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		sync.Mutex{},
		make(map[string]Session),
		make([]Session, 0),
	}
}

func (s *MemoryStorage) Save(sess Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[sess.SessionID()] = sess
	s.list = append(s.list, sess)
	return nil
}

func (s *MemoryStorage) Get(sid string) (Session, error) {
	return s.sessions[sid], nil
}

func (s *MemoryStorage) Rip(sid string) error {
	if _, ok := s.sessions[sid]; ok {
		delete(s.sessions, sid)
		idx := s.indexOf(sid)
		s.list = append(s.list[:idx], s.list[idx+1:]...)
	}
	return nil
}

func (s *MemoryStorage) indexOf(id string) int64 {
	for i, sess := range s.list {
		if sess.SessionID() == id {
			return int64(i)
		}
	}
	return -1
}

func (s *MemoryStorage) Reap(checker AgeChecker) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var marker int
	for i := 0; i < len(s.list); i++ {
		sess := s.list[i]
		if checker.ShouldReap(sess) {
			marker = i
			continue
		}
		break
	}
	for i := 0; i <= marker; i++ {
		delete(s.sessions, s.list[i].SessionID())
	}
	s.list = s.list[marker:]
}
