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
	ct  int64
	at  int64
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

func (c *cache) Add(info *sessionInfo) {
	node := &cacheNode{
		info, 0, nil,
	}
	node.anchor = c.collec.PushBack(node)
	node.sidIdxPos, _ = c.findIndex(info.sid)
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

func (c *cache) Get(sid string) *sessionInfo {
	if found := c.find(sid); found != nil {
		return found.info
	}
	return nil
}

type cacheI interface {
	Add(sess *sessionInfo)
	Contains(sid string) bool
	ExpiredSessions(checker AgeChecker) []string
	Remove(sid string)
	Get(sid string) *sessionInfo
}

var ReservedFields = []string{"ct", "at"}

// Provider that communicates with storage api to init, read and destroy sessions.
type provider struct {
	mu      sync.Mutex
	cached  cacheI
	storage Storage
}

// Returns a new provider (address for pointer reference).
func newProvider(storage Storage) *provider {
	if storage == nil {
		panic("nil storage")
	}
	p := &provider{
		cached: &cache{
			list.New(),
			[]*cacheNode{},
		},
		storage: storage,
	}
	sids, err := storage.List()
	if err != nil {
		panic("unable to load storage sessions")
	}
	for _, sid := range sids {
		data, err := storage.Read(sid)
		if err != nil {
			panic("unable to load storage sessions")
		}
		p.cached.Add(&sessionInfo{
			sid,
			data["ct"].(int64),
			data["at"].(int64),
		})
	}
	return p
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
	now := time.Now().UnixNano()
	sess := &session{
		p,
		sid,
		make(map[string]any),
		now,
		now,
		false,
	}
	p.cached.Add(&sessionInfo{
		sess.id,
		sess.ct,
		sess.at,
	})
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
	if info := p.cached.Get(sid); info != nil {
		return &session{
			p,
			info.sid,
			make(map[string]any),
			info.ct,
			info.at,
			false,
		}, nil
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
	p.mu.Lock()
	defer p.mu.Unlock()
	_sess := sess.(*session)
	got, _ := p.storage.Read(_sess.id)
	for k, v := range _sess.v {
		got[k] = v
	}
	for k, v := range got {
		_sess.v[k] = v
	}
	p.storage.Save(_sess.id, _sess.v)
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
