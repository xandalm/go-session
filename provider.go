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
	ErrEmptySessionId   error = errors.New("session provider: the session id cannot be empty")
	ErrRestoringSession error = errors.New("session provider: cannot restore session from storage")
)

func (p *Provider) SessionInit(sid string) (ISession, error) {
	if sid == "" {
		return nil, ErrEmptySessionId
	}
	sess := p.builder.Build(sid)
	p.storage.Save(sess)
	return sess, nil
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
