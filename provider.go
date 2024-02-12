package session

import "errors"

type SessionBuilder interface {
	Build(sid string) ISession
}

type SessionStorage interface {
	Save(ISession) error
	Get(sid string) (ISession, error)
	Destroy(sid string) error
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
	sess := p.builder.Build(sid)
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
	err := p.storage.Destroy(sid)
	if err != nil {
		return ErrUnableToDestroySession
	}
	return nil
}

func (p *Provider) SessionGC(maxAge int64) {
}
