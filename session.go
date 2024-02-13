package session

type Session struct{}

type SessionBuilder struct{}

func (sb *SessionBuilder) Build(sid string) *Session {
	return &Session{}
}
