package session

type MemoryStorage struct {
	sessions map[string]ISession
}

func (s *MemoryStorage) Save(sess ISession) error {
	s.sessions[sess.SessionID()] = sess
	return nil
}
