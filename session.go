package session

import (
	"errors"
	"maps"
	"reflect"
	"sync"
)

var ErrProtectedKeyName error = errors.New("session: the given key name is protected")

var protectedKeyNames map[string]int8

type session struct {
	mu sync.Mutex
	id string
	v  Values
	fn OnSessionMutation
}

func (s *session) SessionID() string {
	return s.id
}

func (s *session) Get(key string) any {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.v[key]
}

func (s *session) Set(key string, value any) error {
	if _, ok := protectedKeyNames[key]; ok {
		return ErrProtectedKeyName
	}
	rValue := reflect.ValueOf(value)
	for rValue.Kind() == reflect.Pointer {
		rValue = reflect.Indirect(rValue)
	}

	s.mu.Lock()
	s.v[key] = s.mapped(rValue)
	if s.fn != nil {
		s.fn(s)
	}
	s.mu.Unlock()
	return nil
}

func (s *session) mapped(v reflect.Value) any {
	switch v.Kind() {
	case reflect.Func:
		panic("session: cannot stores func into session")
	case reflect.Chan:
		panic("session: cannot stores chan into session")
	case reflect.Struct:
		vFields := reflect.VisibleFields(v.Type())
		m := map[string]any{}
		for _, f := range vFields {
			fValue := v.FieldByName(f.Name)
			if fValue.Kind() == reflect.Struct || fValue.Kind() == reflect.Map {
				m[f.Name] = s.mapped(fValue)
			} else {
				m[f.Name] = fValue.Interface()
			}
		}
		return m
	case reflect.Map:
		m := map[string]any{}
		for _, k := range v.MapKeys() {
			m[k.String()] = s.mapped(v.MapIndex(k))
		}
		return m
	default:
		return v.Interface()
	}
}

func (s *session) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := protectedKeyNames[key]; ok {
		return ErrProtectedKeyName
	}

	delete(s.v, key)
	if s.fn != nil {
		s.fn(s)
	}
	return nil
}

type sessionFactory struct{}

// Create implements SessionFactory.
func (sf *sessionFactory) Create(id string, m Values, fn OnSessionMutation) Session {
	s := &session{
		id: id,
		v:  make(Values),
	}
	for k, v := range m {
		protectedKeyNames[k] = 1
		s.v[k] = v
	}
	return s
}

// Restore implements SessionFactory.
func (sf *sessionFactory) Restore(id string, m Values, v Values, fn OnSessionMutation) Session {
	s := &session{
		id: id,
		v:  make(Values),
	}
	maps.Copy(s.v, v)
	for key, value := range m {
		delete(s.v, key) // meta can't be common
		protectedKeyNames[key] = 1
		s.v[key] = value
	}
	return s
}

// OverrideValues implements SessionFactory.
func (sf *sessionFactory) OverrideValues(sess Session, v Values) {
	_sess := sess.(*session)
	_sess.mu.Lock()
	defer _sess.mu.Unlock()
	for key, value := range v {
		_sess.v[key] = value
	}
}

// ExtractValues implements SessionFactory.
func (sf *sessionFactory) ExportValues(sess Session) Values {
	_sess := sess.(*session)
	_sess.mu.Lock()
	defer _sess.mu.Unlock()
	return maps.Clone(_sess.v)
}

var DefaultSessionFactory SessionFactory = &sessionFactory{}

func init() {
	protectedKeyNames = make(map[string]int8)
}
