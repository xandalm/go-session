package session

import (
	"container/list"
	"errors"
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

type index struct {
	sessions  *list.List // absolute sessions (sessionInfo)
	sidSorted *list.List // sorted by id
	ctSorted  *list.List // serted by creation time
}

func newIndex() *index {
	return &index{
		list.New(),
		list.New(),
		list.New(),
	}
}

func (idx *index) find(sid string) *list.Element {
	elem := idx.sessions.Front()
	for {
		if elem == nil || elem.Value.(*sessionInfo).sid == sid {
			break
		}
		elem = elem.Next()
	}
	return elem
}

func (idx *index) Add(sess *session) {
	idx.sessions.PushBack(&sessionInfo{
		sess.id,
		sess.ct,
		sess.at,
	})
}

func (idx *index) Remove(sid string) {
	found := idx.find(sid)
	idx.remove(found)
}

func (idx *index) remove(e *list.Element) {
	idx.sessions.Remove(e)
}

func (idx *index) Contains(sid string) bool {
	found := idx.find(sid)
	return found != nil
}

func (idx *index) ExpiredSessions(checker AgeChecker) []string {
	var ret []string
	elem := idx.sessions.Front()
	for {
		if elem == nil {
			break
		}
		sess := elem.Value.(*sessionInfo)
		if !checker.ShouldReap(sess.ct) {
			break
		}
		idx.remove(elem)
		ret = append(ret, sess.sid)
		elem = elem.Next()
	}
	return ret
}

// Provider that communicates with storage api to init, read and destroy sessions.
type defaultProvider struct {
	idx               *index
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
		&index{},
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
	contains := p.idx.Contains(sid)
	if contains {
		return nil, ErrDuplicatedSessionId
	}
	sess := &session{
		sid,
		make(map[string]any),
		time.Now(),
		time.Now(),
	}
	p.idx.Add(sess)
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
	for _, sid := range p.idx.ExpiredSessions(p.ageCheckerAdapter(maxAge)) {
		p.storage.Delete(sid)
	}
}
