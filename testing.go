package session

import (
	"errors"
	"sync"
	"time"
)

type stubSession struct {
	Id        string
	CreatedAt int64
	V         map[string]any
}

func (s *stubSession) Set(key string, value any) error {
	s.V[key] = value
	return nil
}

func (s *stubSession) Get(key string) any {
	if key == "ct" {
		return s.CreatedAt
	}
	return s.V[key]
}

func (s *stubSession) Delete(key string) error {
	delete(s.V, key)
	return nil
}

func (s *stubSession) SessionID() string {
	return s.Id
}

type mockSessionFactory struct {
	CreateFunc         func(string, map[string]any) Session
	RestoreFunc        func(string, map[string]any, map[string]any) Session
	OverrideValuesFunc func(Session, map[string]any)
	ExtractValuesFunc  func(Session) map[string]any
}

func (sf *mockSessionFactory) Create(id string, m map[string]any) Session {
	return sf.CreateFunc(id, m)
}

func (sf *mockSessionFactory) Restore(id string, m map[string]any, v map[string]any) Session {
	return sf.RestoreFunc(id, m, v)
}

func (sf *mockSessionFactory) OverrideValues(sess Session, values map[string]any) {
	sf.OverrideValuesFunc(sess, values)
}

func (sf *mockSessionFactory) ExtractValues(sess Session) map[string]any {
	return sf.ExtractValuesFunc(sess)
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

func (p *stubProvider) SessionGC() {}

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

func (p *stubFailingProvider) SessionGC() {}

type stubStorage struct {
	mu   sync.Mutex
	data map[string]map[string]any
}

func newStubStorage() *stubStorage {
	return &stubStorage{
		data: make(map[string]map[string]any),
	}
}

func (ss *stubStorage) Save(id string, values map[string]any) error {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	if ss.data == nil {
		ss.data = map[string]map[string]any{}
	}
	ss.data[id] = values
	return nil
}

func (ss *stubStorage) Read(id string) (map[string]any, error) {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	if v, ok := ss.data[id]; ok {
		return v, nil
	}
	return nil, nil
}

func (ss *stubStorage) List() ([]string, error) {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	ret := []string{}
	for k := range ss.data {
		ret = append(ret, k)
	}
	return ret, nil
}

func (ss *stubStorage) Delete(id string) error {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	delete(ss.data, id)
	return nil
}

var errFoo error = errors.New("foo error")

type stubMilliAgeChecker int64

func (m stubMilliAgeChecker) ShouldReap(t int64) bool {
	diff := time.Now().UnixMilli() - (t / int64(time.Millisecond))
	return diff > int64(m)
}

func NowTime() time.Time {
	return time.Now()
}

func NowTimeNanoseconds() int64 {
	return NowTime().UnixNano()
}
