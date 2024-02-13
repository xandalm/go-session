package session

type Session struct {
	id string
}

func (s *Session) Set(key, value any) error {
	return nil
}

func (s *Session) Get(key any) any {
	return nil
}

func (s *Session) Delete(key any) error {
	return nil
}

func (s *Session) SessionID() string {
	return s.id
}

type SessionBuilder struct{}

func (sb *SessionBuilder) Build(sid string, onSessionUpdate func(sess ISession) error) ISession {
	return &Session{
		id: sid,
	}
}
