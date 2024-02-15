package session

import (
	"errors"
	"time"
)

type ISessionBuilder interface {
	Build(sid string, onSessionUpdate func(ISession) error) ISession
}

type AgeChecker interface {
	ShouldReap(ISession) bool
}

type SessionStorage interface {
	Save(ISession) error
	Get(sid string) (ISession, error)
	Rip(sid string) error
	Reap(AgeChecker)
}

type AgeCheckerAdapter func(int64) AgeChecker

type Provider struct {
	builder           ISessionBuilder
	storage           SessionStorage
	ageCheckerAdapter AgeCheckerAdapter
}

func NewProvider(builder ISessionBuilder, storage SessionStorage, adapter AgeCheckerAdapter) *Provider {
	return &Provider{
		builder,
		storage,
		adapter,
	}
}

var (
	ErrEmptySessionId             error = errors.New("session: the session id cannot be empty")
	ErrRestoringSession           error = errors.New("session: cannot restore session from storage")
	ErrDuplicateSessionId         error = errors.New("session: cannot duplicate session id")
	ErrUnableToEnsureNonDuplicity error = errors.New("session: cannot ensure non duplicity of the sid (storage failure)")
	ErrUnableToDestroySession     error = errors.New("session: unable to destroy session (storage failure)")
	ErrUnableToSaveSession        error = errors.New("session: unable to save session (storage failure)")
)

func (p *Provider) SessionInit(sid string) (ISession, error) {
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

func (p *Provider) ensureNonDuplication(sid string) (bool, error) {
	found, err := p.storage.Get(sid)
	if err != nil {
		return false, err
	}
	return found == nil, nil
}

func (p *Provider) SessionRead(sid string) (ISession, error) {
	sess, err := p.storage.Get(sid)
	if err != nil {
		return nil, ErrRestoringSession
	}
	return sess, nil
}

func (p *Provider) SessionDestroy(sid string) error {
	err := p.storage.Rip(sid)
	if err != nil {
		return ErrUnableToDestroySession
	}
	return nil
}

func (p *Provider) SessionGC(maxAge int64) {
	p.storage.Reap(p.ageCheckerAdapter(maxAge))
}

type SecondsBasedAgeChecker int64

func (ma SecondsBasedAgeChecker) ShouldReap(sess ISession) bool {
	if sess == nil {
		panic("session: cannot check age from nil session")
	}
	diff := time.Now().Unix() - sess.CreationTime().Unix()
	return diff >= int64(ma)
}
