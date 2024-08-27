package session

import (
	"container/list"
	"errors"
	"slices"
	"sync"
	"time"
)

type cacheNode struct {
	sess   Session
	idxPos int
	anchor *list.Element
}

type cache struct {
	collec *list.List   // absolute sessions (sessionInfo)
	idx    []*cacheNode // sorted by id
}

func (c *cache) findIndex(sid string) (int, bool) {
	return slices.BinarySearchFunc(c.idx, sid, func(in *cacheNode, s string) int {
		if in.sess.SessionID() < s {
			return -1
		}
		if in.sess.SessionID() > s {
			return 1
		}
		return 0
	})
}

func (c *cache) find(sid string) *cacheNode {
	pos, has := c.findIndex(sid)
	if has {
		return c.idx[pos]
	}
	return nil
}

func (c *cache) Add(sess Session) {
	node := &cacheNode{
		sess, 0, nil,
	}
	node.anchor = c.collec.PushBack(node)
	node.idxPos, _ = c.findIndex(sess.SessionID())
	c.idx = slices.Insert(c.idx, node.idxPos, node)
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
	c.idx = slices.Delete(c.idx, n.idxPos, n.idxPos+1)
	for i := n.idxPos; i < len(c.idx); i++ {
		c.idx[i].idxPos -= 1
	}
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
		ct := node.sess.Get("ct").(int64)
		if !checker.ShouldReap(ct) {
			break
		}
		c.remove(node)
		ret = append(ret, node.sess.SessionID())
		elem = elem.Next()
	}
	return ret
}

func (c *cache) Get(sid string) Session {
	if found := c.find(sid); found != nil {
		return found.sess
	}
	return nil
}

var ReservedFields = []string{"ct", "at"}

// Provider that communicates with storage api to init, read and destroy sessions.
type provider struct {
	mu sync.Mutex     // Mutex
	ac AgeChecker     // Expiration checker
	ca *cache         // Cached sessions
	st Storage        // Session storage (persistence, normally)
	sf SessionFactory // Session factory for session analysis
}

var providerSyncTimer *time.Timer // must be stopped on new provider creation

func interruptProviderSyncRoutine() {
	if providerSyncTimer != nil {
		providerSyncTimer.Stop()
		providerSyncTimer = nil
	}
}

// Returns a new provider (address for pointer reference).
func newProvider(ac AgeChecker, sf SessionFactory, storage Storage) *provider {
	interruptProviderSyncRoutine()
	p := &provider{
		ac: ac,
		ca: &cache{
			list.New(),
			[]*cacheNode{},
		},
		sf: sf,
		st: storage,
	}
	sids, err := storage.List()
	if err != nil {
		panic("session: unable to load storage sessions")
	}
	for _, sid := range sids {
		data, err := storage.Read(sid)
		if err != nil {
			panic("session: unable to load storage sessions")
		}
		meta := map[string]any{
			"ct": data["ct"],
		}
		delete(data, "ct")
		p.ca.Add(p.sf.Restore(sid, meta, data))
	}
	p.storageSync()
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
	if p.ca.Contains(sid) {
		return nil, ErrDuplicatedSessionId
	}
	now := time.Now().UnixNano()
	sess := p.sf.Create(sid, map[string]any{"ct": now})
	p.ca.Add(sess)
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
	if sess := p.ca.Get(sid); sess != nil {
		if !p.ac.ShouldReap(sess.Get("ct").(int64)) {
			return sess, nil
		}
		p.ca.Remove(sid)
	}
	return p.sessionInit(sid)
}

// Destroys the session.
//
// Returns error when cannot remove through storage api.
func (p *provider) SessionDestroy(sid string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.ca.Remove(sid)
	if p.st != nil {
		p.st.Delete(sid)
	}
	return nil
}

// Checks for expired sessions through storage api, and remove them.
// The maxAge will be adapted accordingly to AgeCheckerAdapter
func (p *provider) SessionGC() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, sid := range p.ca.ExpiredSessions(p.ac) {
		if p.st != nil {
			p.st.Delete(sid)
		}
	}
}

var ProviderSyncRoutineTime time.Duration = 10 * time.Second

func (p *provider) storageSync() {
	p.mu.Lock()
	defer p.mu.Unlock()
	elem := p.ca.collec.Front()
	for {
		if elem == nil {
			break
		}
		sess := elem.Value.(*cacheNode).sess
		p.st.Save(sess.SessionID(), p.sf.ExtractValues(sess))
		elem = elem.Next()
	}
	providerSyncTimer = time.AfterFunc(ProviderSyncRoutineTime, func() {
		p.storageSync()
	})
}
