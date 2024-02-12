package session

import "errors"

type SessionBuilder interface {
	Build(sid string) ISession
}

type SessionStorage interface {
	Save(ISession) error
	Get(sid string) (ISession, error)
}

type Provider struct {
	builder SessionBuilder
	storage SessionStorage
}

func NewProvider(builder SessionBuilder, storage SessionStorage) *Provider {
	return &Provider{
		builder,
		storage,
	}
}

var (
	ErrEmptySessionId           error = errors.New("session provider: the session id cannot be empty")
	ErrRestoringSession         error = errors.New("session provider: cannot restore session from storage")
	ErrDuplicateSessionId       error = errors.New("session provider: cannot duplicate session id")
	ErrCannotEnsureNonDuplicity error = errors.New("session provider: cannot ensure non duplicity of the sid (storage failing)")
)

func (p *Provider) SessionInit(sid string) (ISession, error) {
	if sid == "" {
		return nil, ErrEmptySessionId
	}
	ok, err := p.ensureNonDuplication(sid)
	if err != nil {
		return nil, ErrCannotEnsureNonDuplicity
	}
	if !ok {
		return nil, ErrDuplicateSessionId
	}
	sess := p.builder.Build(sid)
	p.storage.Save(sess)
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
	return nil
}

func (p *Provider) SessionGC(maxAge int64) {
}
