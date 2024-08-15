package session

import (
	"fmt"
	"reflect"
	"slices"
	"sync"
)

type session struct {
	mu   sync.Mutex
	p    Provider
	id   string
	v    map[string]any
	ct   int64
	at   int64
	sync bool
}

func (s *session) SessionID() string {
	return s.id
}

func (s *session) Get(key string) any {
	s.mu.Lock()
	defer s.mu.Unlock()
	if got, ok := s.v[key]; ok {
		return got
	}
	if !s.sync && s.p.SessionSync(s) != nil {
		return nil
	}
	s.sync = true
	return s.v[key]
}

func (s *session) Set(key string, value any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if slices.Contains(ReservedFields, key) {
		panic(fmt.Sprintf("sorry, you can't use any from %v as key", ReservedFields))
	}
	rValue := reflect.ValueOf(value)
	for rValue.Kind() == reflect.Pointer {
		rValue = reflect.Indirect(rValue)
	}
	s.v[key] = s.mapped(rValue)
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
	if _, ok := s.v[key]; ok {
		delete(s.v, key)
		return
	}
	if !s.sync && s.p.SessionSync(s) != nil {
		return
	}
	s.sync = true
	delete(s.v, key)
}
