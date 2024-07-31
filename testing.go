package session

import (
	"errors"
	"maps"
	"sync"
	"time"
)

type stubSession struct {
	Id        string
	CreatedAt time.Time
	V         map[string]any
}

func newStubSession(id string) *stubSession {
	return &stubSession{
		Id:        id,
		CreatedAt: time.Now(),
		V:         map[string]any{},
	}
}

func (s *stubSession) Set(key string, value any) error {
	s.V[key] = value
	return nil
}

func (s *stubSession) Get(key string) any {
	return s.V[key]
}

func (s *stubSession) Delete(key string) error {
	delete(s.V, key)
	return nil
}

func (s *stubSession) Values() map[string]any {
	return maps.Clone(s.V)
}

func (s *stubSession) SessionID() string {
	return s.Id
}

func (s *stubSession) CreationTime() time.Time {
	return s.CreatedAt
}

type stubProvider struct {
	mu       sync.Mutex
	Sessions map[string]Session
}

func (p *stubProvider) SessionInit(sid string) (Session, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.Sessions == nil {
		p.Sessions = make(map[string]Session)
	}
	sess := &stubSession{
		Id: sid,
	}
	p.Sessions[sid] = sess
	return sess, nil
}

func (p *stubProvider) SessionRead(sid string) (Session, error) {
	sess := p.Sessions[sid]
	return sess, nil
}

func (p *stubProvider) SessionDestroy(sid string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.Sessions, sid)
	return nil
}

func (p *stubProvider) SessionGC(checker AgeChecker) {}

type stubFailingProvider struct{}

func (p *stubFailingProvider) SessionInit(sid string) (Session, error) {
	return nil, errFoo
}

func (p *stubFailingProvider) SessionRead(sid string) (Session, error) {
	return nil, errFoo
}

func (p *stubFailingProvider) SessionDestroy(sid string) error {
	return errFoo
}

func (p *stubFailingProvider) SessionGC(checker AgeChecker) {}

type stubSessionStorage struct {
	mu       sync.Mutex
	Sessions map[string]Session
}

func newStubSessionStorage() *stubSessionStorage {
	return &stubSessionStorage{
		Sessions: make(map[string]Session),
	}
}

func (ss *stubSessionStorage) Save(sess Session) error {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	ss.Sessions[sess.SessionID()] = sess
	return nil
}

func (ss *stubSessionStorage) Load(sid string) (Session, error) {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	if sess, ok := ss.Sessions[sid]; ok {
		return sess, nil
	}
	return nil, nil
}

func (ss *stubSessionStorage) Delete(sid string) error {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	delete(ss.Sessions, sid)
	return nil
}

// func (ss *stubSessionStorage) CreateSession(sid string) (Session, error) {
// 	ss.mu.Lock()
// 	defer ss.mu.Unlock()
// 	sess := newStubSession(sid)
// 	ss.Sessions[sid] = sess
// 	return sess, nil
// }

// func (ss *stubSessionStorage) GetSession(sid string) (Session, error) {
// 	if sess, ok := ss.Sessions[sid]; ok {
// 		return sess, nil
// 	}
// 	return nil, nil
// }

// func (ss *stubSessionStorage) ContainsSession(sid string) (bool, error) {
// 	_, ok := ss.Sessions[sid]
// 	return ok, nil
// }

// func (ss *stubSessionStorage) ReapSession(sid string) error {
// 	ss.mu.Lock()
// 	defer ss.mu.Unlock()
// 	delete(ss.Sessions, sid)
// 	return nil
// }

// func (ss *stubSessionStorage) Deadline(checker AgeChecker) {
// 	ss.mu.Lock()
// 	defer ss.mu.Unlock()
// 	for k, v := range ss.Sessions {
// 		if checker.ShouldReap(v.CreatedAt) {
// 			delete(ss.Sessions, k)
// 		}
// 	}
// }

type spySessionStorage struct {
	callsToSave   int
	callsToLoad   int
	callsToDelete int
	// callsToCreateSession   int
	// callsToGetSession      int
	// callsToContainsSession int
	// callsToReapSession     int
	// callsToDeadline        int
}

func (ss *spySessionStorage) Save(sess Session) error {
	ss.callsToSave++
	return nil
}

func (ss *spySessionStorage) Load(sid string) (Session, error) {
	ss.callsToLoad++
	return nil, nil
}

func (ss *spySessionStorage) Delete(sid string) error {
	ss.callsToDelete++
	return nil
}

// func (ss *spySessionStorage) CreateSession(sid string) (Session, error) {
// 	ss.callsToCreateSession++
// 	return nil, nil
// }

// func (ss *spySessionStorage) GetSession(sid string) (Session, error) {
// 	ss.callsToGetSession++
// 	return nil, nil
// }

// func (ss *spySessionStorage) ContainsSession(sid string) (bool, error) {
// 	ss.callsToContainsSession++
// 	return false, nil
// }

// func (ss *spySessionStorage) ReapSession(sid string) error {
// 	ss.callsToReapSession++
// 	return nil
// }

// func (ss *spySessionStorage) Deadline(checker AgeChecker) {
// 	ss.callsToDeadline++
// }

var errFoo error = errors.New("foo error")

type stubFailingSessionStorage struct {
	Sessions map[string]Session
}

func (ss *stubFailingSessionStorage) Save(sess Session) error {
	return errFoo
}

func (ss *stubFailingSessionStorage) Load(sid string) (Session, error) {
	return nil, errFoo
}

func (ss *stubFailingSessionStorage) Delete(sid string) error {
	return errFoo
}

// func (ss *stubFailingSessionStorage) CreateSession(sid string) (Session, error) {
// 	return nil, errFoo
// }

// func (ss *stubFailingSessionStorage) GetSession(sid string) (Session, error) {
// 	return nil, errFoo
// }

// func (ss *stubFailingSessionStorage) ContainsSession(sid string) (bool, error) {
// 	return false, errFoo
// }

// func (ss *stubFailingSessionStorage) ReapSession(sid string) error {
// 	return errFoo
// }

// func (ss *stubFailingSessionStorage) Deadline(checker AgeChecker) {
// }

type mockSessionStorage struct {
	Sessions            map[string]Session
	CreateSessionFunc   func(sid string) (Session, error)
	GetSessionFunc      func(sid string) (Session, error)
	ContainsSessionFunc func(sid string) (bool, error)
	ReapSessionFunc     func(sid string) error
	DeadlineFunc        func(checker AgeChecker)
}

func (ss *mockSessionStorage) CreateSession(sid string) (Session, error) {
	return ss.CreateSessionFunc(sid)
}

func (ss *mockSessionStorage) GetSession(sid string) (Session, error) {
	return ss.GetSessionFunc(sid)
}

func (ss *mockSessionStorage) ContainsSession(sid string) (bool, error) {
	return ss.ContainsSessionFunc(sid)
}

func (ss *mockSessionStorage) ReapSession(sid string) error {
	return ss.ReapSessionFunc(sid)
}

func (ss *mockSessionStorage) Deadline(checker AgeChecker) {
	ss.DeadlineFunc(checker)
}

type stubMilliAgeChecker int64

func (m stubMilliAgeChecker) ShouldReap(t time.Time) bool {
	diff := time.Now().UnixMilli() - t.UnixMilli()
	return diff > int64(m)
}

func stubMilliAgeCheckerAdapter(v int64) AgeChecker {
	return stubMilliAgeChecker(v)
}

type stubCache map[string]Session

func (c stubCache) Add(sess Session) {
	c[sess.SessionID()] = sess
}

func (c stubCache) Contains(sid string) bool {
	_, ok := c[sid]
	return ok
}

func (c stubCache) ExpiredSessions(checker AgeChecker) []string {
	return nil
}

func (c stubCache) Remove(sid string) {
	delete(c, sid)
}

func (c stubCache) Get(sid string) Session {
	if g, ok := c[sid]; ok {
		return g
	}
	return nil
}

type mockCache struct {
	AddFunc             func(Session)
	ContainsFunc        func(string) bool
	ExpiredSessionsFunc func(AgeChecker) []string
	RemoveFunc          func(string)
	GetFunc             func(string) Session
}

func (c *mockCache) Add(sess Session) {
	c.AddFunc(sess)
}

func (c *mockCache) Contains(sid string) bool {
	return c.ContainsFunc(sid)
}

func (c *mockCache) ExpiredSessions(checker AgeChecker) []string {
	return c.ExpiredSessionsFunc(checker)
}

func (c *mockCache) Remove(sid string) {
	c.RemoveFunc(sid)
}

func (c *mockCache) Get(sid string) Session {
	return c.GetFunc(sid)
}

type spyCache struct {
	callsToAdd             int
	callsToContains        int
	callsToExpiredSessions int
	callsToRemove          int
	callsToGet             int
}

func (c *spyCache) Add(sess Session) {
	c.callsToAdd++
}

func (c *spyCache) Contains(sid string) bool {
	c.callsToContains++
	return false
}

func (c *spyCache) ExpiredSessions(checker AgeChecker) []string {
	c.callsToExpiredSessions++
	return nil
}

func (c *spyCache) Remove(sid string) {
	c.callsToRemove++
}

func (c *spyCache) Get(sid string) Session {
	c.callsToGet++
	return nil
}
