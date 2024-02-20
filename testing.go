package session

import (
	"errors"
	"sync"
	"time"
)

type stubSession struct {
	Id        string
	CreatedAt time.Time
	OnUpdate  func(Session) error
}

func newStubSession(id string, t time.Time, onUpdate func(Session) error) *stubSession {
	return &stubSession{
		Id:        id,
		CreatedAt: t,
		OnUpdate:  onUpdate,
	}
}

func (s *stubSession) Set(key string, value any) error {
	s.OnUpdate(s)
	return nil
}

func (s *stubSession) Get(key string) any {
	s.OnUpdate(s)
	return nil
}

func (s *stubSession) Delete(key string) error {
	s.OnUpdate(s)
	return nil
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

type stubSessionBuilder struct {
}

func (sb *stubSessionBuilder) Build(sid string, onSessionUpdate func(Session) error) Session {
	return &stubSession{
		Id:        sid,
		CreatedAt: time.Now(),
		OnUpdate:  onSessionUpdate,
	}
}

func (sb *stubSessionBuilder) Restore(sid string, creationTime time.Time, values SessionValues, onSessionUpdate func(Session) error) (Session, error) {
	return &stubSession{
		Id:        sid,
		CreatedAt: creationTime,
		OnUpdate:  onSessionUpdate,
	}, nil
}

type stubSessionStorage struct {
	mu       sync.Mutex
	Sessions map[string]Session
}

func (ss *stubSessionStorage) Save(sess Session) error {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	if ss.Sessions == nil {
		ss.Sessions = make(map[string]Session)
	}
	ss.Sessions[sess.SessionID()] = sess
	return nil
}

func (ss *stubSessionStorage) Get(sid string) (Session, error) {
	sess := ss.Sessions[sid]
	return sess, nil
}

func (ss *stubSessionStorage) Rip(sid string) error {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	delete(ss.Sessions, sid)
	return nil
}

func (ss *stubSessionStorage) Reap(checker AgeChecker) {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	for k, v := range ss.Sessions {
		if checker.ShouldReap(v) {
			delete(ss.Sessions, k)
		}
	}
}

type spySessionBuilder struct {
	callsToBuild   int
	callsToRestore int
}

func (sb *spySessionBuilder) Build(sid string, onSessionUpdate func(Session) error) Session {
	sb.callsToBuild++
	return nil
}

func (sb *spySessionBuilder) Restore(sid string, creationTime time.Time, values SessionValues, onSessionUpdate func(Session) error) (Session, error) {
	sb.callsToRestore++
	return nil, nil
}

type spySessionStorage struct {
	callsToSave int
	callsToGet  int
	callsToRip  int
	callsToReap int
}

func (ss *spySessionStorage) Save(sess Session) error {
	ss.callsToSave++
	return nil
}

func (ss *spySessionStorage) Get(sid string) (Session, error) {
	ss.callsToGet++
	return nil, nil
}

func (ss *spySessionStorage) Rip(sid string) error {
	ss.callsToRip++
	return nil
}

func (ss *spySessionStorage) Reap(checker AgeChecker) {
	ss.callsToReap++
}

type stubFailingSessionStorage struct {
	Sessions map[string]Session
}

var errFoo error = errors.New("foo error")

func (ss *stubFailingSessionStorage) Save(sess Session) error {
	return errFoo
}

func (ss *stubFailingSessionStorage) Get(sid string) (Session, error) {
	return nil, errFoo
}

func (ss *stubFailingSessionStorage) Rip(sid string) error {
	return errFoo
}

func (ss *stubFailingSessionStorage) Reap(checker AgeChecker) {
}

type mockSessionStorage struct {
	Sessions map[string]Session
	SaveFunc func(sess Session) error
	GetFunc  func(sid string) (Session, error)
	RipFunc  func(sid string) error
	ReapFunc func(checker AgeChecker)
}

func (ss *mockSessionStorage) Save(sess Session) error {
	return ss.SaveFunc(sess)
}

func (ss *mockSessionStorage) Get(sid string) (Session, error) {
	return ss.GetFunc(sid)
}

func (ss *mockSessionStorage) Rip(sid string) error {
	return ss.RipFunc(sid)
}

func (ss *mockSessionStorage) Reap(checker AgeChecker) {
	ss.ReapFunc(checker)
}

type stubNanoAgeChecker int64

func (m stubNanoAgeChecker) ShouldReap(sess Session) bool {
	diff := time.Now().UnixNano() - sess.CreationTime().UnixNano()
	return diff > int64(m)
}

type stubMilliAgeChecker int64

func (m stubMilliAgeChecker) ShouldReap(sess Session) bool {
	diff := time.Now().UnixMilli() - sess.CreationTime().UnixMilli()
	return diff > int64(m)
}
