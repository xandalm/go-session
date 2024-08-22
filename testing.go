package session

import (
	"errors"
	"maps"
	"sync"
	"time"
)

type stubSession struct {
	Id        string
	CreatedAt int64
	V         map[string]any
}

func newStubSession(id string) *stubSession {
	return &stubSession{
		Id:        id,
		CreatedAt: time.Now().UnixNano(),
		V:         map[string]any{},
	}
}

func (s *stubSession) Set(key string, value any) {
	s.V[key] = value
}

func (s *stubSession) Get(key string) any {
	if key == "ct" {
		return s.CreatedAt
	}
	return s.V[key]
}

func (s *stubSession) Delete(key string) {
	delete(s.V, key)
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

func (p *stubProvider) SessionPush(sess Session) error {
	return nil
}

func (p *stubProvider) SessionPull(sess Session) error {
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

func (p *stubFailingProvider) SessionPush(sess Session) error {
	return errFoo
}

func (p *stubFailingProvider) SessionPull(sess Session) error {
	return errFoo
}

func (p *stubFailingProvider) SessionGC(checker AgeChecker) {}

type spyProvider struct {
	callsToInit    int
	callsToRead    int
	callsToDestroy int
	callsToSync    int
	callsToPull    int
	callsToGC      int
}

func (p *spyProvider) SessionInit(sid string) (Session, error) {
	p.callsToInit++
	return nil, nil
}

func (p *spyProvider) SessionRead(sid string) (Session, error) {
	p.callsToRead++
	return nil, nil
}

func (p *spyProvider) SessionDestroy(sid string) error {
	p.callsToDestroy++
	return nil
}

func (p *spyProvider) SessionPush(sess Session) error {
	p.callsToSync++
	return nil
}

func (p *spyProvider) SessionPull(sess Session) error {
	p.callsToPull++
	return nil
}

func (p *spyProvider) SessionGC(checker AgeChecker) {
	p.callsToGC++
}

type mockProvider struct {
	SessionInitFunc    func(sid string) (Session, error)
	SessionReadFunc    func(sid string) (Session, error)
	SessionDestroyFunc func(sid string) error
	SessionSyncFunc    func(sess Session) error
	SessionGCFunc      func(checker AgeChecker)
}

func (p *mockProvider) SessionInit(sid string) (Session, error) {
	return p.SessionInitFunc(sid)
}

func (p *mockProvider) SessionRead(sid string) (Session, error) {
	return p.SessionReadFunc(sid)
}

func (p *mockProvider) SessionDestroy(sid string) error {
	return p.SessionDestroyFunc(sid)
}

func (p *mockProvider) SessionPush(sess Session) error {
	return p.SessionSyncFunc(sess)
}

func (p *mockProvider) SessionPull(sess Session) error {
	return p.SessionSyncFunc(sess)
}

func (p *mockProvider) SessionGC(checker AgeChecker) {
	p.SessionGCFunc(checker)
}

type stubStorageItem struct {
	id     string
	values map[string]any
}

func (r *stubStorageItem) Id() string {
	return r.id
}

func (r *stubStorageItem) Set(k string, v any) {
	r.values[k] = v
}

func (r *stubStorageItem) Delete(k string) {
	delete(r.values, k)
}

func (r *stubStorageItem) Values() map[string]any {
	return maps.Clone(r.values)
}

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
	for k, _ := range ss.data {
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

type spyStorage struct {
	callsToSave   int
	callsToLoad   int
	callsToDelete int
}

func (ss *spyStorage) Save(r Session) error {
	ss.callsToSave++
	return nil
}

func (ss *spyStorage) Load(id string) (Session, error) {
	ss.callsToLoad++
	return nil, nil
}

func (ss *spyStorage) Delete(id string) error {
	ss.callsToDelete++
	return nil
}

var errFoo error = errors.New("foo error")

// type stubFailingStorage struct {
// }

// func (s *stubFailingStorage) Save(r StorageItem) error {
// 	return errFoo
// }

// func (s *stubFailingStorage) Load(id string) (StorageItem, error) {
// 	return nil, errFoo
// }

// func (s *stubFailingStorage) Delete(id string) error {
// 	return errFoo
// }

// type mockStorage struct {
// 	SaveFunc   func(StorageItem) error
// 	LoadFunc   func(string) (StorageItem, error)
// 	DeleteFunc func(string) error
// }

// func (s *mockStorage) Save(r StorageItem) error {
// 	return s.SaveFunc(r)
// }

// func (s *mockStorage) Load(id string) (StorageItem, error) {
// 	return s.LoadFunc(id)
// }

// func (s *mockStorage) Delete(id string) error {
// 	return s.DeleteFunc(id)
// }

type stubMilliAgeChecker int64

func (m stubMilliAgeChecker) ShouldReap(t int64) bool {
	diff := time.Now().UnixMilli() - (t / int64(time.Millisecond))
	return diff > int64(m)
}

func stubMilliAgeCheckerAdapter(v int64) AgeChecker {
	return stubMilliAgeChecker(v)
}

// type mockCache struct {
// 	AddFunc             func(Session)
// 	ContainsFunc        func(string) bool
// 	ExpiredSessionsFunc func(AgeChecker) []string
// 	RemoveFunc          func(string)
// 	GetFunc             func(string) Session
// }

// func (c *mockCache) Add(sess Session) {
// 	c.AddFunc(sess)
// }

// func (c *mockCache) Contains(sid string) bool {
// 	return c.ContainsFunc(sid)
// }

// func (c *mockCache) ExpiredSessions(checker AgeChecker) []string {
// 	return c.ExpiredSessionsFunc(checker)
// }

// func (c *mockCache) Remove(sid string) {
// 	c.RemoveFunc(sid)
// }

// func (c *mockCache) Get(sid string) Session {
// 	return c.GetFunc(sid)
// }

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

func NowTime() time.Time {
	return time.Now()
}

func NowTimeNanoseconds() int64 {
	return NowTime().UnixNano()
}
