package session

import (
	"errors"
	"time"
)

type stubSession struct {
	Id        string
	CreatedAt time.Time
	OnUpdate  func(ISession) error
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

type stubProvider struct {
	Sessions map[string]ISession
}

func (p *stubProvider) SessionInit(sid string) (ISession, error) {
	if p.Sessions == nil {
		p.Sessions = make(map[string]ISession)
	}
	sess := &stubSession{
		Id: sid,
	}
	p.Sessions[sid] = sess
	return sess, nil
}

func (p *stubProvider) SessionRead(sid string) (ISession, error) {
	sess := p.Sessions[sid]
	return sess, nil
}

func (p *stubProvider) SessionDestroy(sid string) error {
	delete(p.Sessions, sid)
	return nil
}

func (p *stubProvider) SessionGC(maxLifeTime int64) {}

type stubSessionBuilder struct {
}

func (sb *stubSessionBuilder) Build(sid string, onSessionUpdate func(ISession) error) ISession {
	return &stubSession{
		Id:       sid,
		OnUpdate: onSessionUpdate,
	}
}

type stubSessionStorage struct {
	Sessions map[string]ISession
}

func (ss *stubSessionStorage) Save(sess ISession) error {
	if ss.Sessions == nil {
		ss.Sessions = make(map[string]ISession)
	}
	ss.Sessions[sess.SessionID()] = sess
	return nil
}

func (ss *stubSessionStorage) Get(sid string) (ISession, error) {
	sess := ss.Sessions[sid]
	return sess, nil
}

func (ss *stubSessionStorage) Rip(sid string) error {
	delete(ss.Sessions, sid)
	return nil
}

func (ss *stubSessionStorage) Reap(maxAge int64) {
	for k, v := range ss.Sessions {
		diff := time.Now().Unix() - v.(*stubSession).CreatedAt.Unix()
		if diff >= maxAge {
			delete(ss.Sessions, k)
		}
	}
}

type stubFailingSessionStorage struct {
	Sessions map[string]ISession
}

var ErrFoo error = errors.New("foo error")

func (ss *stubFailingSessionStorage) Save(sess ISession) error {
	return ErrFoo
}

func (ss *stubFailingSessionStorage) Get(sid string) (ISession, error) {
	return nil, ErrFoo
}

func (ss *stubFailingSessionStorage) Rip(sid string) error {
	return ErrFoo
}

func (ss *stubFailingSessionStorage) Reap(maxAge int64) {
}

type mockSessionStorage struct {
	Sessions map[string]ISession
	SaveFunc func(sess ISession) error
	GetFunc  func(sid string) (ISession, error)
	RipFunc  func(sid string) error
	ReapFunc func(maxAge int64)
}

func (ss *mockSessionStorage) Save(sess ISession) error {
	return ss.SaveFunc(sess)
}

func (ss *mockSessionStorage) Get(sid string) (ISession, error) {
	return ss.GetFunc(sid)
}

func (ss *mockSessionStorage) Rip(sid string) error {
	return ss.RipFunc(sid)
}

func (ss *mockSessionStorage) Reap(maxAge int64) {
	ss.ReapFunc(maxAge)
}
