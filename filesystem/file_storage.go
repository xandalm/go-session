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

func (s *session) Get(key string) any {
	return s.v[key]
}
