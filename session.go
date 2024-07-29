package session

import (
	"reflect"
	"time"
)

type session struct {
	id string
	v  map[string]any
	ct time.Time
	at time.Time
}

func (s *session) SessionID() string {
	return s.id
}

func (s *session) Get(key string) any {
	return s.v[key]
}

func (s *session) Set(key string, value any) error {
	rValue := reflect.ValueOf(value)
	for rValue.Kind() == reflect.Pointer {
		rValue = reflect.Indirect(rValue)
	}
	s.v[key] = s.mapped(rValue)
	return nil
}

func (s *session) mapped(v reflect.Value) any {
	switch v.Kind() {
	case reflect.Func:
		panic("cannot stores func into session")
	case reflect.Chan:
		panic("cannot stores chan into session")
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
	delete(s.v, key)
	return nil
}
