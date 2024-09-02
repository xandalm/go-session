package session

import (
	"container/list"
	"context"
	"errors"
	"slices"
	"sync"
	"time"
)

type sessionInfo struct {
	sess Session
	id   string
	ct   int64
}

type cacheNode struct {
	info   *sessionInfo
	idxPos int
	anchor *list.Element
}

type cache struct {
	collec *list.List   // absolute sessions (sessionInfo)
	idx    []*cacheNode // sorted by id
}

func (c *cache) findIndex(sid string) (int, bool) {
	return slices.BinarySearchFunc(c.idx, sid, func(in *cacheNode, s string) int {
		if in.info.id < s {
			return -1
		}
		if in.info.id > s {
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

func (c *cache) Add(info *sessionInfo) {
	n := &cacheNode{
		info, 0, nil,
	}
	n.anchor = c.collec.PushBack(n)
	n.idxPos, _ = c.findIndex(info.id)
	c.idx = slices.Insert(c.idx, n.idxPos, n)
	for i := n.idxPos + 1; i < len(c.idx); i++ {
		c.idx[i].idxPos += 1
	}
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

func (c *cache) ExpiredSessions(checker ageChecker) []string {
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
		ret = append(ret, node.info.id)
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

var ReservedFields = []string{"ct", "at"}

// Provider that communicates with storage api to init, read and destroy sessions.
type provider struct {
	mu sync.Mutex     // Mutex
	ac ageChecker     // Expiration checker
	ca *cache         // Cached sessions
	st Storage        // Session storage (persistence, normally)
	sf SessionFactory // Session factory for session analysis
	// t  *time.Timer
}

var _p *provider = nil

// func (p *provider) interruptSyncRoutine() {
// 	if p == nil || p.t == nil {
// 		return
// 	}
// 	p.mu.Lock()
// 	defer p.mu.Unlock()
// 	p.t.Stop()
// 	p.t = nil
// }

// Returns a new provider (address for pointer reference).
func newProvider(ac ageChecker, sf SessionFactory, storage Storage) *provider {
	// if _p != nil {
	// 	_p.interruptSyncRoutine()
	// }
	_p = &provider{
		ac: ac,
		ca: &cache{
			list.New(),
			[]*cacheNode{},
		},
		sf: sf,
		st: storage,
	}
	if storage == nil {
		return _p
	}
	sids, err := storage.List()
	if err != nil {
		panic("session: unable to load storage sessions")
	}
	for _, sid := range sids {
		data, err := storage.Read(sid)
		if err != nil {
			storage.Delete(sid)
			continue
		}
		meta := map[string]any{
			"ct": data["ct"],
		}
		// delete(data, "ct")

		sess := _p.sf.Restore(sid, meta, nil)

		_p.ca.Add(&sessionInfo{
			sess,
			sess.SessionID(),
			sess.Get("ct").(int64),
		})
	}
	// _p.storageSync()
	return _p
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
func (p *provider) SessionInit(ctx context.Context, sid string) (Session, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.sessionInit(ctx, sid)
}

func (p *provider) sessionInit(ctx context.Context, sid string) (Session, error) {
	if sid == "" {
		return nil, ErrEmptySessionId
	}
	if p.ca.Contains(sid) {
		return nil, ErrDuplicatedSessionId
	}
	now := time.Now().UnixNano()
	sess := p.sf.Create(sid, map[string]any{"ct": now})
	info := &sessionInfo{
		sess,
		sid,
		now,
	}
	p.ca.Add(info)

	p.registerSessionPush(ctx, info)

	return sess, nil
}

// Restores the session accordingly to given session identifier. If
// the session does not exists, then will create through SessionInit().
//
// Returns error when cannot get session through storage api, or cannot
// create one. Otherwise, will return the session.
func (p *provider) SessionRead(ctx context.Context, sid string) (Session, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	info := p.ca.Get(sid)
	if info == nil {
		return p.sessionInit(ctx, sid)
	}
	if p.ac.ShouldReap(info.ct) {
		p.ca.Remove(sid)
		p.st.Delete(sid)
		return p.sessionInit(ctx, sid)
	}
	if info.sess == nil {
		data, err := p.st.Read(sid)
		if err != nil {
			p.ca.Remove(sid)
			return p.sessionInit(ctx, sid)
		}
		meta := map[string]any{
			"ct": info.ct,
		}
		delete(data, "ct")
		info.sess = p.sf.Restore(info.id, meta, data)
	}
	p.registerSessionPush(ctx, info)
	return info.sess, nil
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

	expired := p.ca.ExpiredSessions(p.ac)
	if p.st == nil {
		return
	}
	for _, sid := range expired {
		p.st.Delete(sid)
	}
}

func (p *provider) registerSessionPush(ctx context.Context, info *sessionInfo) {
	go push(ctx, p, info)
}

func push(ctx context.Context, p *provider, info *sessionInfo) {
	<-ctx.Done()
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.ca.Contains(info.id) {
		return
	}
	sess := info.sess
	if sess == nil {
		return
	}
	p.st.Save(sess.SessionID(), p.sf.ExtractValues(sess))
	info.sess = nil
}

// var ProviderSyncRoutineTime time.Duration = 10 * time.Second

// func (p *provider) storageSync() {
// 	p.mu.Lock()
// 	defer p.mu.Unlock()
// 	elem := p.ca.collec.Front()
// 	for {
// 		if elem == nil {
// 			break
// 		}
// 		sess := elem.Value.(*cacheNode).sess
// 			p.st.Save(sess.SessionID(), p.sf.ExtractValues(sess))
// 		elem = elem.Next()
// 	}
// 	p.t = time.AfterFunc(ProviderSyncRoutineTime, func() {
// 		p.storageSync()
// 	})
// }
