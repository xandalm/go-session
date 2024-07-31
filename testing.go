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

type stubStorage struct {
	mu   sync.Mutex
	data map[string]StorageItem
}

func newStubStorage() *stubStorage {
	return &stubStorage{
		data: make(map[string]StorageItem),
	}
}

func (ss *stubStorage) Save(r StorageItem) error {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	ss.data[r.Id()] = r
	return nil
}

func (ss *stubStorage) Load(id string) (StorageItem, error) {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	if sess, ok := ss.data[id]; ok {
		return sess, nil
	}
	return nil, nil
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

func (ss *spyStorage) Save(r StorageItem) error {
	ss.callsToSave++
	return nil
}

func (ss *spyStorage) Load(id string) (StorageItem, error) {
	ss.callsToLoad++
	return nil, nil
}

func (ss *spyStorage) Delete(id string) error {
	ss.callsToDelete++
	return nil
}

var errFoo error = errors.New("foo error")

type stubFailingStorage struct {
}

func (s *stubFailingStorage) Save(r StorageItem) error {
	return errFoo
}

func (s *stubFailingStorage) Load(id string) (StorageItem, error) {
	return nil, errFoo
}

func (s *stubFailingStorage) Delete(id string) error {
	return errFoo
}

type mockStorage struct {
	SaveFunc   func(StorageItem) error
	LoadFunc   func(string) (StorageItem, error)
	DeleteFunc func(string) error
}

func (s *mockStorage) Save(r StorageItem) error {
	return s.SaveFunc(r)
}

func (s *mockStorage) Load(id string) (StorageItem, error) {
	return s.LoadFunc(id)
}

func (s *mockStorage) Delete(id string) error {
	return s.DeleteFunc(id)
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
