package session

type MemoryStorage struct {
	sessions map[string]ISession
}

func (s *MemoryStorage) Save(sess ISession) error {
	if sid := sess.SessionID(); sid != "" {
		s.sessions[sess.SessionID()] = sess
		return nil
	}
	return ErrEmptySessionId
}
