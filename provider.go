package session

type SessionBuilder interface {
	Build(sid string) ISession
}

type SessionStorage interface {
	Save(ISession) error
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

func (p *Provider) SessionInit(sid string) (ISession, error) {
	sess := p.builder.Build(sid)
	p.storage.Save(sess)
	return sess, nil
}

func (p *Provider) SessionRead(sid string) (ISession, error) {
	return nil, nil
}

func (p *Provider) SessionDestroy(sid string) error {
	return nil
}

func (p *Provider) SessionGC(maxAge int64) {
}
