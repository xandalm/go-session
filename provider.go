package session

import (
	"container/list"
	"errors"
	"slices"
	"sync"
	"time"
)

type sessionInfo struct {
	sid string
	ct  time.Time
	at  time.Time
}

type cacheNode struct {
	info      *sessionInfo
	sidIdxPos int
	anchor    *list.Element
}

type cache struct {
	collec *list.List   // absolute sessions (sessionInfo)
	sidIdx []*cacheNode // sorted by id
}

func (c *cache) findIndex(sid string) (int, bool) {
	return slices.BinarySearchFunc(c.sidIdx, sid, func(in *cacheNode, s string) int {
		if in.info.sid < s {
			return -1
		}
		if in.info.sid > s {
			return 1
		}
		return 0
	})
}

func (c *cache) find(sid string) *cacheNode {
	pos, has := c.findIndex(sid)
	if has {
		return c.sidIdx[pos]
	}
	return nil
}

func (c *cache) Add(sess Session) {
	s := sess.(*session)
	node := &cacheNode{
		&sessionInfo{
			s.id,
			s.ct,
			s.at,
		}, 0, nil,
	}
	node.anchor = c.collec.PushBack(node)
	node.sidIdxPos, _ = c.findIndex(s.id)
	c.sidIdx = slices.Insert(c.sidIdx, node.sidIdxPos, node)
}

func (c *cache) Remove(sid string) {
	found := c.find(sid)
	if found == nil {
		return
	}
	c.remove(found)
}

func (c *cache) remove(n *cacheNode) {
	c.collec.Remove(n.anchor)
	c.sidIdx = slices.Delete(c.sidIdx, n.sidIdxPos, n.sidIdxPos+1)
}

func (c *cache) Contains(sid string) bool {
	_, has := c.findIndex(sid)
	return has
}

func (c *cache) ExpiredSessions(checker AgeChecker) []string {
	var ret []string
	elem := c.collec.Front()
	for {
		if elem == nil {
			break
		}
		node := elem.Value.(*cacheNode)
		if !checker.ShouldReap(node.info.ct) {
			break
		}
		c.remove(node)
		ret = append(ret, node.info.sid)
		elem = elem.Next()
	}
	return ret
}

func (c *cache) Get(sid string) Session {
	if found := c.find(sid); found != nil {
		return &session{
			id: found.info.sid,
			ct: found.info.ct,
			at: found.info.at,
		}
	}
	return nil
}

type cacheI interface {
	Add(sess Session)
	Contains(sid string) bool
	ExpiredSessions(checker AgeChecker) []string
	Remove(sid string)
	Get(sid string) Session
}

// Provider that communicates with storage api to init, read and destroy sessions.
type provider struct {
	mu      sync.Mutex
	cached  cacheI
	storage Storage
	s2i     Session2StorageItem
	i2s     StorageItem2Session
}

// Returns a new provider (address for pointer reference).
func newProvider(storage Storage) *provider {
	if storage == nil {
		panic("nil storage")
	}
	return &provider{
		cached: &cache{
			list.New(),
			[]*cacheNode{},
		},
		storage: storage,
	}
}

var (
	ErrEmptySessionId             error = errors.New("session: sid cannot be empty")
	ErrDuplicatedSessionId        error = errors.New("session: cannot duplicate sid")
	ErrUnableToRestoreSession     error = errors.New("session: unable to restore session (storage failure)")
	ErrUnableToEnsureNonDuplicity error = errors.New("session: unable to ensure non-duplicity of sid (storage failure)")
	ErrUnableToDestroySession     error = errors.New("session: unable to destroy session (storage failure)")
	ErrUnableToSaveSession        error = errors.New("session: unable to save session (storage failure)")
)

// Creates a session with the given session identifier.
//
// Returns an error when:
// - The identifier cannot be empty;
// - Cannot check if the identifier is already in use;
// - The given identifier already exists;
// - Session cannot be created through the storage api.
//
// Otherwise, will return the session.
func (p *provider) SessionInit(sid string) (Session, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.sessionInit(sid)
}

func (p *provider) sessionInit(sid string) (Session, error) {
	if sid == "" {
		return nil, ErrEmptySessionId
	}
	if p.cached.Contains(sid) {
		return nil, ErrDuplicatedSessionId
	}
	sess := &session{
		p,
		sid,
		make(map[string]any),
		time.Now(),
		time.Now(),
	}
	p.cached.Add(sess)
	return sess, nil
}

// Restores the session accordingly to given session identifier. If
// the session does not exists, then will create through SessionInit().
//
// Returns error when cannot get session through storage api, or cannot
// create one. Otherwise, will return the session.
func (p *provider) SessionRead(sid string) (Session, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if sess := p.cached.Get(sid); sess != nil {
		return sess, nil
	}
	return p.sessionInit(sid)
}

// Destroys the session.
//
// Returns error when cannot remove through storage api.
func (p *provider) SessionDestroy(sid string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.cached.Remove(sid)
	p.storage.Delete(sid)
	return nil
}

func (p *provider) SessionSync(sess Session) error {
	got, _ := p.storage.Load(sess.SessionID(), p.i2s)
	for k, v := range sess.values() {
		got.values()[k] = v
	}
	for k, v := range got.values() {
		sess.values()[k] = v
	}
	p.storage.Save(sess, p.s2i)
	return nil
}

// Checks for expired sessions through storage api, and remove them.
// The maxAge will be adapted accordingly to AgeCheckerAdapter
func (p *provider) SessionGC(checker AgeChecker) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, sid := range p.cached.ExpiredSessions(checker) {
		p.storage.Delete(sid)
	}
}
