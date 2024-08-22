package session

import (
	"fmt"
	"maps"
	"reflect"
	"slices"
	"sync"
)

type session struct {
	mu sync.Mutex
	id string
	v  map[string]any
}

func (s *session) SessionID() string {
	return s.id
}

func (s *session) Get(key string) any {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.v[key]
}

func (s *session) Set(key string, value any) {
	if slices.Contains(ReservedFields, key) {
		panic(fmt.Sprintf("sorry, you can't use any from %v as key", ReservedFields))
	}
	rValue := reflect.ValueOf(value)
	for rValue.Kind() == reflect.Pointer {
		rValue = reflect.Indirect(rValue)
	}

	s.mu.Lock()
	s.v[key] = s.mapped(rValue)
	s.mu.Unlock()
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

func (s *session) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.v, key)
}

type sessionFactory struct{}

// Create implements SessionFactory.
func (sf *sessionFactory) Create(id string, m map[string]any) Session {
	s := &session{
		id: id,
		v:  make(map[string]any),
	}
	maps.Copy(s.v, m)
	return s
}

// ExtractValues implements SessionFactory.
func (sf *sessionFactory) ExtractValues(Session) map[string]any {
	panic("unimplemented")
}

// OverrideValues implements SessionFactory.
func (sf *sessionFactory) OverrideValues(Session, map[string]any) {
	panic("unimplemented")
}

// Restore implements SessionFactory.
func (sf *sessionFactory) Restore(id string, m map[string]any, v map[string]any) Session {
	panic("unimplemented")
}

func NewSessionFactory() SessionFactory {
	return &sessionFactory{}
}
