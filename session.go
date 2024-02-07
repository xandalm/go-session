package session

type Session interface {
	Set(key, value any) error
	Get(key any) any
	Delete(key any) error
	SessionID() string
}

type Provider interface {
	SessionInit(sid string) (Session, error)
	SessionRead(sid string) (Session, error)
	SessionDestroy(sid string) error
	SessionGC(maxLifeTime int64)
}

type Manager struct {
	provider    Provider
	cookieName  string
	maxLifeTime int64
}

func NewManager(provider Provider, cookieName string, maxLifeTime int64) *Manager {
	return &Manager{
		provider:    provider,
		cookieName:  cookieName,
		maxLifeTime: maxLifeTime,
	}
}
