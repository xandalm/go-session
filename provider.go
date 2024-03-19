package session

import (
	"errors"
	"time"
)

type AgeChecker interface {
	ShouldReap(time.Time) bool
}

type Storage interface {
	CreateSession(sid string) (Session, error)
	GetSession(sid string) (Session, error)
	ContainsSession(sid string) (bool, error)
	ReapSession(sid string) error
	Deadline(AgeChecker)
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

// Provider that communicates with storage api to init, read and destroy sessions.
type defaultProvider struct {
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
	contains, err := p.storage.ContainsSession(sid)

	if err != nil {
		return nil, ErrUnableToEnsureNonDuplicity
	}
	if contains {
		return nil, ErrDuplicatedSessionId
	}
	sess, err := p.storage.CreateSession(sid)
	if err != nil {
		return nil, ErrUnableToSaveSession
	}
	return sess, nil
}

// Restores the session accordingly to given session identifier. If
// the session does not exists, then will create through SessionInit().
//
// Returns error when cannot get session through storage api, or cannot
// create one. Otherwise, will return the session.
func (p *defaultProvider) SessionRead(sid string) (Session, error) {
	sess, err := p.storage.GetSession(sid)
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
	err := p.storage.ReapSession(sid)
	if err != nil {
		return ErrUnableToDestroySession
	}
	return nil
}

// Checks for expired sessions through storage api, and remove them.
// The maxAge will be adapted accordingly to AgeCheckerAdapter
func (p *defaultProvider) SessionGC(maxAge int64) {
	p.storage.Deadline(p.ageCheckerAdapter(maxAge))
}
