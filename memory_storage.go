package session

import (
	"sync"
	"time"
)

type registry struct {
	id        string
	createdAt time.Time
	values    map[string]any
}

func newRegistry(id string, createdAt time.Time, values map[string]any) registry {
	return registry{id, createdAt, values}
}

type MemoryStorage struct {
	mu             sync.Mutex
	sessions       map[string]registry
	list           []registry
	sessionBuilder SessionBuilder
}

func NewMemoryStorage(sessionBuilder SessionBuilder) *MemoryStorage {
	return &MemoryStorage{
		sync.Mutex{},
		make(map[string]registry),
		make([]registry, 0),
		sessionBuilder,
	}
}

func (s *MemoryStorage) Save(sess Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	reg := newRegistry(sess.SessionID(), sess.CreationTime(), sess.Values())
	s.sessions[sess.SessionID()] = reg
	s.list = append(s.list, reg)
	return nil
}

func (s *MemoryStorage) Get(sid string) (Session, error) {
	if reg, ok := s.sessions[sid]; ok {
		sess, err := s.sessionBuilder.Restore(reg.id, reg.createdAt, reg.values, s.Save)
		if err != nil {
			return nil, err
		}
		return sess, nil
	}
	return nil, nil
}

func (s *MemoryStorage) Rip(sid string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.sessions[sid]; ok {
		delete(s.sessions, sid)
		idx := s.indexOf(sid)
		s.list = append(s.list[:idx], s.list[idx+1:]...)
	}
	return nil
}

func (s *MemoryStorage) indexOf(id string) int64 {
	for i, reg := range s.list {
		if reg.id == id {
			return int64(i)
		}
	}
	return -1
}

func (s *MemoryStorage) Reap(checker AgeChecker) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var marker int = -1
	for i := 0; i < len(s.list); i++ {
		reg := s.list[i]
		if checker.ShouldReap(reg.createdAt) {
			marker = i
			continue
		}
		break
	}
	if marker == -1 {
		return
	}
	for i := 0; i <= marker; i++ {
		reg := s.list[i]
		delete(s.sessions, reg.id)
	}
	s.list = s.list[marker:]
}
