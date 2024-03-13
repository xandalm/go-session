package filesystem

import (
	"encoding/gob"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
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
	s.v[key] = s.mapped(rValue)
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
	delete(s.v, key)
	return nil
}

type storageIO interface {
	Create(sid string) (*session, error)
	Read(sid string) (*session, error)
	Write(sess *session) error
	Delete(sid string) error
	List() []string
}

type defaultStorageIO struct {
	path string
	ext  string
	mu   sync.Mutex
}

func newStorageIO(path, dir, ext string) *defaultStorageIO {
	path, err := filepath.Abs(path)
	if err == nil {
		path = filepath.Join(path, dir)
		err = os.MkdirAll(path, 0750)
		if err == nil || os.IsExist(err) {
			return &defaultStorageIO{
				path,
				ext,
				sync.Mutex{},
			}
		}
	}
	panic(fmt.Errorf("session: cannot make sessions storage folder, %v", err))
}

func (sio *defaultStorageIO) create(w io.Writer, sess *session) error {
	sess.ct = time.Now()
	return sio.write(w, sess)
}

func (sio *defaultStorageIO) Create(sid string) (*session, error) {
	sio.mu.Lock()
	defer sio.mu.Unlock()

	file, err := os.OpenFile(sio.filePath(sid), os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	sess := &session{id: sid, v: map[string]any{}}
	if err := sio.create(file, sess); err != nil {
		return nil, err
	}
	return sess, nil
}

func (sio *defaultStorageIO) read(r io.Reader) (*session, error) {

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

func (sio *defaultStorageIO) Read(sid string) (*session, error) {
	sio.mu.Lock()
	defer sio.mu.Unlock()

	file, err := os.Open(sio.filePath(sid))
	if err != nil {
		return nil, err
	}
	defer file.Close()
	sess, err := sio.read(file)
	if err != nil {
		return nil, err
	}
	sess.id = sid
	return sess, nil
}

func (sio *defaultStorageIO) write(w io.Writer, sess *session) error {
	enc := gob.NewEncoder(w)

	sess.at = time.Now()

	err := enc.Encode(&extSession{
		Ct: sess.ct.UnixNano(),
		At: sess.at.UnixNano(),
		V:  sess.v,
	})
	return err
}

func (sio *defaultStorageIO) Write(sess *session) error {
	sio.mu.Lock()
	defer sio.mu.Unlock()

	file, err := os.OpenFile(sio.filePath(sess.id), os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Seek(0, 0)
	if err != nil {
		return err
	}
	return sio.write(file, sess)
}

func (sio *defaultStorageIO) Delete(sid string) error {
	sio.mu.Lock()
	defer sio.mu.Unlock()

	return os.Remove(sio.filePath(sid))
}

func (sio *defaultStorageIO) List() []string {
	names := []string{}
	entries, err := os.ReadDir(sio.path)
	if err != nil {
		return nil
	}
	for _, entry := range entries {
		names = append(names, strings.TrimSuffix(entry.Name(), "."+sio.ext))
	}

	return names
}

func (sio *defaultStorageIO) filePath(sid string) string {
	return filepath.Join(sio.path, sid+"."+sio.ext)
}

type storage struct {
	io storageIO
}

func NewStorage(path, dir, ext string) *storage {
	io := newStorageIO(path, dir, ext)
	return &storage{io}
}

func (s *storage) CreateSession(sid string) (sessionpkg.Session, error) {
	sess, err := s.io.Create(sid)
	if err != nil {
		return nil, err
	}
	return sess, nil
}

func (s *storage) GetSession(sid string) (sessionpkg.Session, error) {
	sess, err := s.io.Read(sid)
	if err != nil {
		return nil, err
	}
	return sess, nil
}

func (s *storage) ReapSession(sid string) error {
	return s.io.Delete(sid)
}

func (s *storage) Deadline(checker sessionpkg.AgeChecker) {
	for _, name := range s.io.List() {
		sess, _ := s.io.Read(name)
		if sess == nil {
			continue
		}
		if !checker.ShouldReap(sess.ct) {
			break
		}
		s.io.Delete(name)
	}
}

func init() {
	gob.Register(map[string]any{})
}
