package session

import (
	"sync"
	"time"
)

type MemoryStorage struct {
	mu       sync.Mutex
	sessions map[string]ISession
	list     []ISession
}

func NewMemoryStorage(sessions map[string]ISession, list []ISession) *MemoryStorage {
	return &MemoryStorage{
		sync.Mutex{},
		sessions,
		list,
	}
}

func (s *MemoryStorage) Save(sess ISession) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[sess.SessionID()] = sess
	s.list = append(s.list, sess)
	return nil
}

func (s *MemoryStorage) Get(sid string) (ISession, error) {
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

func (s *MemoryStorage) Reap(maxAge int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var marker int
	for i := 0; i < len(s.list); i++ {
		sess := s.list[i]
		if time.Now().Unix()-sess.CreationTime().Unix() >= maxAge {
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
