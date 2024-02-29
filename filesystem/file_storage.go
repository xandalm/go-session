package filesystem

import "time"

type session struct {
	id string
	v  map[string]any
	ct time.Time
}

func (s *session) SessionID() string {
	return s.id
}
