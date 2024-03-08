package filesystem

import (
	"encoding/gob"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"time"

	sessionpkg "github.com/xandalm/go-session"
)

type session struct {
	id string
	v  map[string]any
	ct time.Time
	at time.Time
}

type extSession struct {
	V      map[string]any
	Ct, At int64
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
	s.v[key] = s.walk(rValue)
	return nil
}

func (s *session) walk(v reflect.Value) any {
	switch v.Kind() {
	case reflect.Struct:
		vFields := reflect.VisibleFields(v.Type())
		m := map[string]any{}
		for _, f := range vFields {
			fValue := v.FieldByName(f.Name)
			if fValue.Kind() == reflect.Struct || fValue.Kind() == reflect.Map {
				m[f.Name] = s.walk(fValue)
			} else {
				m[f.Name] = fValue.Interface()
			}
		}
		return m
	case reflect.Map:
		m := map[string]any{}
		for _, k := range v.MapKeys() {
			m[k.String()] = s.walk(v.MapIndex(k))
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

type storage struct {
	path string
	ext  string
}

func NewStorage(path, dir, ext string) *storage {
	path, err := filepath.Abs(path)
	if err == nil {
		path = filepath.Join(path, dir)
		err = os.MkdirAll(path, 0750)
		if err == nil || os.IsExist(err) {
			return &storage{path, ext}
		}
	}
	panic("session: cannot make sessions storage folder")
}

func (s *storage) CreateSession(sid string) (sessionpkg.Session, error) {
	file, err := os.OpenFile(s.filePath(sid), os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	sess := &session{id: sid, v: map[string]any{}}
	if err := s.createSession(file, sess); err != nil {
		return nil, err
	}
	return sess, nil
}

func (s *storage) GetSession(sid string) (sessionpkg.Session, error) {
	file, err := os.Open(s.filePath(sid))
	if err != nil {
		return nil, err
	}
	defer file.Close()
	sess, err := s.readSession(file)
	if err != nil {
		return nil, err
	}
	sess.id = sid
	return sess, nil
}

func (s *storage) readSession(r io.Reader) (*session, error) {

	var esess extSession

	dec := gob.NewDecoder(r)

	err := dec.Decode(&esess)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return &session{
		ct: time.Unix(0, esess.Ct),
		at: time.Unix(0, esess.At),
		v:  esess.V,
	}, nil
}

func (s *storage) createSession(w io.Writer, sess *session) error {

	enc := gob.NewEncoder(w)

	now := time.Now()
	sess.ct = now
	sess.at = now

	err := enc.Encode(&extSession{
		Ct: now.UnixNano(),
		At: now.UnixNano(),
	})
	return err
}

func (s storage) filePath(sid string) string {
	return filepath.Join(s.path, sid+"."+s.ext)
}

func init() {
	gob.Register(map[string]any{})
}
