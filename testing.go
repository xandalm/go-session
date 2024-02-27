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

func (p *stubProvider) SessionGC(maxLifeTime int64) {}

type stubSessionStorage struct {
	mu       sync.Mutex
	Sessions map[string]*stubSession
}

func newStubSessionStorage() *stubSessionStorage {
	return &stubSessionStorage{
		Sessions: make(map[string]*stubSession),
	}
}

func (ss *stubSessionStorage) CreateSession(sid string) (Session, error) {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	sess := newStubSession(sid)
	ss.Sessions[sid] = sess
	return sess, nil
}

func (ss *stubSessionStorage) GetSession(sid string) (Session, error) {
	if sess, ok := ss.Sessions[sid]; ok {
		return sess, nil
	}
	return nil, nil
}

func (ss *stubSessionStorage) ReapSession(sid string) error {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	delete(ss.Sessions, sid)
	return nil
}

func (ss *stubSessionStorage) Deadline(checker AgeChecker) {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	for k, v := range ss.Sessions {
		if checker.ShouldReap(v.CreatedAt) {
			delete(ss.Sessions, k)
		}
	}
}

type spySessionStorage struct {
	callsToCreateSession int
	callsToGetSession    int
	callsToReapSession   int
	callsToDeadline      int
}

func (ss *spySessionStorage) CreateSession(sid string) (Session, error) {
	ss.callsToCreateSession++
	return nil, nil
}

func (ss *spySessionStorage) GetSession(sid string) (Session, error) {
	ss.callsToGetSession++
	return nil, nil
}

func (ss *spySessionStorage) ReapSession(sid string) error {
	ss.callsToReapSession++
	return nil
}

func (ss *spySessionStorage) Deadline(checker AgeChecker) {
	ss.callsToDeadline++
}

type stubFailingSessionStorage struct {
	Sessions map[string]Session
}

var errFoo error = errors.New("foo error")

func (ss *stubFailingSessionStorage) CreateSession(sid string) (Session, error) {
	return nil, errFoo
}

func (ss *stubFailingSessionStorage) GetSession(sid string) (Session, error) {
	return nil, errFoo
}

func (ss *stubFailingSessionStorage) ReapSession(sid string) error {
	return errFoo
}

func (ss *stubFailingSessionStorage) Deadline(checker AgeChecker) {
}

type mockSessionStorage struct {
	Sessions          map[string]Session
	CreateSessionFunc func(sid string) (Session, error)
	GetSessionFunc    func(sid string) (Session, error)
	ReapSessionFunc   func(sid string) error
	DeadlineFunc      func(checker AgeChecker)
}

func (ss *mockSessionStorage) CreateSession(sid string) (Session, error) {
	return ss.CreateSessionFunc(sid)
}

func (ss *mockSessionStorage) GetSession(sid string) (Session, error) {
	return ss.GetSessionFunc(sid)
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
