package session

import (
	"container/list"
	"errors"
	"slices"
	"time"
)

type Session interface {
	SessionID() string
	Set(string, any) error
	Get(string) any
	Delete(string) error
}

type Storage interface {
	Save(Session) error
	Load(sid string) (Session, error)
	Delete(sid string) error
	// CreateSession(sid string) (Session, error)
	// GetSession(sid string) (Session, error)
	// ContainsSession(sid string) (bool, error)
	// ReapSession(sid string) error
	// Deadline(AgeChecker)
}

type Provider interface {
	SessionInit(sid string)
	SessionRead(sid string)
	SessionDestroy(sid string)
	SessionGC(maxAge int64)
	Storage()
}

type AgeChecker interface {
	ShouldReap(time.Time) bool
}

type AgeCheckerAdapter func(int64) AgeChecker

type secondsAgeChecker int64

func (ma secondsAgeChecker) ShouldReap(t time.Time) bool {
	diff := time.Now().Unix() - t.Unix()
	return diff >= int64(ma)
}

var SecondsAgeCheckerAdapter AgeCheckerAdapter = func(maxAge int64) AgeChecker {
	return secondsAgeChecker(maxAge)
}

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

func newCache() *cache {
	return &cache{
		list.New(),
		[]*cacheNode{},
	}
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

func (c *cache) Add(sess *session) {
	node := &cacheNode{
		&sessionInfo{
			sess.id,
			sess.ct,
			sess.at,
		}, 0, nil,
	}
	node.anchor = c.collec.PushBack(node)
	node.sidIdxPos, _ = c.findIndex(sess.id)
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

// Provider that communicates with storage api to init, read and destroy sessions.
type defaultProvider struct {
	cached            *cache
	storage           Storage
	ageCheckerAdapter AgeCheckerAdapter
}

// Returns a new defaultProvider (address for pointer reference).
func NewProvider(storage Storage, adapter AgeCheckerAdapter) *defaultProvider {
	if storage == nil {
		panic("nil storage")
	}
	if adapter == nil {
		adapter = SecondsAgeCheckerAdapter
	}
	return &defaultProvider{
		&cache{},
		storage,
		adapter,
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
func (p *defaultProvider) SessionInit(sid string) (Session, error) {
	if sid == "" {
		return nil, ErrEmptySessionId
	}
	contains := p.cached.Contains(sid)
	if contains {
		return nil, ErrDuplicatedSessionId
	}
	sess := &session{
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
func (p *defaultProvider) SessionRead(sid string) (Session, error) {
	sess, err := p.storage.Load(sid)
	if err != nil {
		return nil, ErrUnableToRestoreSession
	}
	if sess == nil {
		sess, err = p.SessionInit(sid)
		return sess, err
	}
	return sess, nil
}

// Destroys the session.
//
// Returns error when cannot remove through storage api.
func (p *defaultProvider) SessionDestroy(sid string) error {
	err := p.storage.Delete(sid)
	if err != nil {
		return ErrUnableToDestroySession
	}
	return nil
}

// Checks for expired sessions through storage api, and remove them.
// The maxAge will be adapted accordingly to AgeCheckerAdapter
func (p *defaultProvider) SessionGC(maxAge int64) {
	for _, sid := range p.cached.ExpiredSessions(p.ageCheckerAdapter(maxAge)) {
		p.storage.Delete(sid)
	}
}
