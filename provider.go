package session

import (
	"errors"
	"time"
)

type SessionBuilder interface {
	Build(sid string, onSessionUpdate func(Session) error) Session
	Expose(sess Session) map[string]any
}

type AgeChecker interface {
	ShouldReap(Session) bool
}

type Storage interface {
	Save(Session) error
	Get(sid string) (Session, error)
	Rip(sid string) error
	Reap(AgeChecker)
}

type AgeCheckerAdapter func(int64) AgeChecker

type secondsAgeChecker int64

func (ma secondsAgeChecker) ShouldReap(sess Session) bool {
	if sess == nil {
		panic("session: cannot check age from nil session")
	}
	diff := time.Now().Unix() - sess.CreationTime().Unix()
	return diff >= int64(ma)
}

var SecondsAgeCheckerAdapter AgeCheckerAdapter = func(maxAge int64) AgeChecker {
	return secondsAgeChecker(maxAge)
}

type DefaultProvider struct {
	builder           SessionBuilder
	storage           Storage
	ageCheckerAdapter AgeCheckerAdapter
}

func NewDefaultProvider(builder SessionBuilder, storage Storage, adapter AgeCheckerAdapter) *DefaultProvider {
	if adapter == nil {
		adapter = SecondsAgeCheckerAdapter
	}
	return &DefaultProvider{
		builder,
		storage,
		adapter,
	}
}

var (
	ErrEmptySessionId             error = errors.New("session: sid(session id) cannot be empty")
	ErrDuplicateSessionId         error = errors.New("session: cannot duplicate sid(session id)")
	ErrUnableToRestoreSession     error = errors.New("session: unable to restore session (storage failure)")
	ErrUnableToEnsureNonDuplicity error = errors.New("session: unable to ensure non-duplicity of sid (storage failure)")
	ErrUnableToDestroySession     error = errors.New("session: unable to destroy session (storage failure)")
	ErrUnableToSaveSession        error = errors.New("session: unable to save session (storage failure)")
)

func (p *DefaultProvider) SessionInit(sid string) (Session, error) {
	if sid == "" {
		return nil, ErrEmptySessionId
	}
	ok, err := p.ensureNonDuplication(sid)
	if err != nil {
		return nil, ErrUnableToEnsureNonDuplicity
	}
	if !ok {
		return nil, ErrDuplicateSessionId
	}
	sess := p.builder.Build(sid, p.storage.Save)
	if err := p.storage.Save(sess); err != nil {
		return nil, ErrUnableToSaveSession
	}
	return sess, nil
}

func (p *DefaultProvider) ensureNonDuplication(sid string) (bool, error) {
	found, err := p.storage.Get(sid)
	if err != nil {
		return false, err
	}
	return found == nil, nil
}

func (p *DefaultProvider) SessionRead(sid string) (Session, error) {
	sess, err := p.storage.Get(sid)
	if err != nil {
		return nil, ErrUnableToRestoreSession
	}
	if sess == nil {
		sess, err = p.SessionInit(sid)
		return sess, err
	}
	return sess, nil
}

func (p *DefaultProvider) SessionDestroy(sid string) error {
	err := p.storage.Rip(sid)
	if err != nil {
		return ErrUnableToDestroySession
	}
	return nil
}

func (p *DefaultProvider) SessionGC(maxAge int64) {
	p.storage.Reap(p.ageCheckerAdapter(maxAge))
}
