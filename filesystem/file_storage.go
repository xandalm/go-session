package filesystem

import (
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// type extSession struct {
// 	V      map[string]any
// 	Ct, At int64
// }

// type session struct {
// 	id string
// 	v  map[string]any
// 	ct time.Time
// 	at time.Time
// }

// func (s *session) SessionID() string {
// 	return s.id
// }

// func (s *session) Get(key string) any {
// 	return s.v[key]
// }

// func (s *session) Set(key string, value any) error {
// 	rValue := reflect.ValueOf(value)
// 	for rValue.Kind() == reflect.Pointer {
// 		rValue = reflect.Indirect(rValue)
// 	}
// 	s.v[key] = s.mapped(rValue)
// 	if err := _storage.update(s); err != nil {
// 		return err
// 	}
// 	return nil
// }

// func (s *session) mapped(v reflect.Value) any {
// 	switch v.Kind() {
// 	case reflect.Func:
// 		panic("cannot stores func into session")
// 	case reflect.Chan:
// 		panic("cannot stores chan into session")
// 	case reflect.Struct:
// 		vFields := reflect.VisibleFields(v.Type())
// 		m := map[string]any{}
// 		for _, f := range vFields {
// 			fValue := v.FieldByName(f.Name)
// 			if fValue.Kind() == reflect.Struct || fValue.Kind() == reflect.Map {
// 				m[f.Name] = s.mapped(fValue)
// 			} else {
// 				m[f.Name] = fValue.Interface()
// 			}
// 		}
// 		return m
// 	case reflect.Map:
// 		m := map[string]any{}
// 		for _, k := range v.MapKeys() {
// 			m[k.String()] = s.mapped(v.MapIndex(k))
// 		}
// 		return m
// 	default:
// 		return v.Interface()
// 	}
// }

// func (s *session) Delete(key string) error {
// 	delete(s.v, key)
// 	if err := _storage.update(s); err != nil {
// 		return err
// 	}
// 	return nil
// }

// type basicSessionInfo struct {
// 	id string
// 	ct int64
// }

// type storageIO interface {
// 	Create(sid string) (*session, error)
// 	Read(sid string) (*session, error)
// 	Write(sess *session) error
// 	Delete(sid string) error
// 	List() []string
// }

// type defaultStorageIO struct {
// 	path   string
// 	prefix string
// }

// func newStorageIO(path string) *defaultStorageIO {
// 	path, err := filepath.Abs(path)
// 	if err == nil {
// 		err = os.MkdirAll(path, 0750)
// 		if err == nil || os.IsExist(err) {
// 			return &defaultStorageIO{
// 				path,
// 				"gosess_",
// 			}
// 		}
// 	}
// 	panic(fmt.Sprintf("cannot make sessions storage folder, %v", err))
// }

// func (sio *defaultStorageIO) create(w io.Writer, v values) error {
// 	return sio.write(w, v)
// }

// func (sio *defaultStorageIO) Create(id string) (values, error) {
// 	file, err := os.OpenFile(sio.filePath(id), os.O_RDWR|os.O_CREATE, 0666)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer file.Close()
// 	sess := &session{id: sid, v: map[string]any{}}
// 	if err := sio.create(file, sess); err != nil {
// 		return nil, err
// 	}
// 	return sess, nil
// }

// func (sio *defaultStorageIO) read(r io.Reader) (*session, error) {

// 	var esess extSession

// 	dec := gob.NewDecoder(r)

// 	err := dec.Decode(&esess)
// 	if err != nil && err != io.EOF {
// 		return nil, err
// 	}

// 	return &session{
// 		ct: time.Unix(0, esess.Ct),
// 		at: time.Unix(0, esess.At),
// 		v:  esess.V,
// 	}, nil
// }

// func (sio *defaultStorageIO) Read(sid string) (*session, error) {
// 	file, err := os.Open(sio.filePath(sid))
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer file.Close()
// 	sess, err := sio.read(file)
// 	if err != nil {
// 		return nil, err
// 	}
// 	sess.id = sid
// 	return sess, nil
// }

// func (sio *defaultStorageIO) write(w io.Writer, sess *session) error {
// 	enc := gob.NewEncoder(w)

// 	sess.at = time.Now()

// 	err := enc.Encode(&extSession{
// 		Ct: sess.ct.UnixNano(),
// 		At: sess.at.UnixNano(),
// 		V:  sess.v,
// 	})
// 	return err
// }

// func (sio *defaultStorageIO) Write(sess *session) error {
// 	file, err := os.OpenFile(sio.filePath(sess.id), os.O_WRONLY, 0666)
// 	if err != nil {
// 		return err
// 	}
// 	defer file.Close()
// 	_, err = file.Seek(0, 0)
// 	if err != nil {
// 		return err
// 	}
// 	return sio.write(file, sess)
// }

// func (sio *defaultStorageIO) Delete(sid string) error {
// 	return os.Remove(sio.filePath(sid))
// }

// func (sio *defaultStorageIO) List() (names []string) {
// 	entries, err := os.ReadDir(sio.path)
// 	if err != nil {
// 		return
// 	}
// 	names = []string{}
// 	for _, entry := range entries {
// 		names = append(names, strings.TrimPrefix(entry.Name(), sio.prefix))
// 	}
// 	return
// }

// func (sio *defaultStorageIO) filePath(sid string) string {
// 	return filepath.Join(sio.path, sio.prefix+sid)
// }

type storage struct {
	// io storageIO
	// m    map[string]*list.Element
	// list *list.List
	mu     sync.Mutex
	path   string
	prefix string
}

func NewStorage(path, prefix string /* io storageIO */) *storage {
	err := os.MkdirAll(path, 0750)
	if err != nil {
		panic(fmt.Sprintf("cannot make sessions storage folder, %v", err))
	}
	s := &storage{
		// io:   io,
		// m:    map[string]*list.Element{},
		// list: list.New(),
		mu:     sync.Mutex{},
		path:   path,
		prefix: prefix,
	}

	// names := s.io.List()
	// if names == nil {
	// 	panic("cannot list sessions files")
	// }

	// // load sessions from file system
	// var hold *list.Element
	// for _, name := range names {
	// 	sess, err := s.io.Read(name)
	// 	if err != nil {
	// 		panic("cannot load sessions files")
	// 	}
	// 	bsi := &basicSessionInfo{
	// 		sess.id,
	// 		sess.ct.UnixNano(),
	// 	}
	// 	for hold = s.list.Back(); hold != nil; hold = hold.Prev() {
	// 		hbsi := hold.Value.(*basicSessionInfo)
	// 		if bsi.ct >= hbsi.ct {
	// 			break
	// 		}
	// 		s.m[hbsi.id] = s.list.InsertAfter(hold.Value, hold)
	// 	}
	// 	if hold == nil {
	// 		s.m[bsi.id] = s.list.PushBack(bsi)
	// 	} else {
	// 		hold.Value = bsi
	// 		s.m[bsi.id] = hold
	// 	}
	// }
	return s
}

func (s *storage) Save(id string, values map[string]any) error {

	file, err := os.OpenFile(filepath.Join(s.path, s.prefix+id), os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer file.Close()
	enc := gob.NewEncoder(file)

	if err := enc.Encode(values); err != nil {
		return err
	}
	return nil
}

// // Returns a session or an error if cannot creates a session and it's file.
// func (s *storage) CreateSession(sid string) (sessionpkg.Session, error) {
// 	s.mu.Lock()
// 	defer s.mu.Unlock()

// 	sess, err := s.io.Create(sid)
// 	if err != nil {
// 		return nil, err
// 	}
// 	s.m[sid] = s.list.PushBack(&basicSessionInfo{
// 		sess.id,
// 		sess.ct.UnixNano(),
// 	})
// 	return sess, nil
// }

// // Returns a session or an error if cannot reads the session from it's file.
// func (s *storage) GetSession(sid string) (sessionpkg.Session, error) {
// 	s.mu.Lock()
// 	defer s.mu.Unlock()

// 	if _, ok := s.m[sid]; ok {
// 		sess, err := s.io.Read(sid)
// 		if err != nil {
// 			return nil, err
// 		}
// 		return sess, nil
// 	}
// 	return nil, nil
// }

// // Checks if the storage contains the session.
// func (s *storage) ContainsSession(sid string) (bool, error) {
// 	s.mu.Lock()
// 	defer s.mu.Unlock()

// 	_, ok := s.m[sid]
// 	return ok, nil
// }

// // Destroys the session from the storage and it's file.
// func (s *storage) ReapSession(sid string) error {
// 	s.mu.Lock()
// 	defer s.mu.Unlock()

// 	if elem, ok := s.m[sid]; ok {
// 		if err := s.io.Delete(sid); err != nil {
// 			return err
// 		}
// 		s.list.Remove(elem)
// 		delete(s.m, sid)
// 	}
// 	return nil
// }

// // Scans the storage removing expired sessions.
// func (s *storage) Deadline(checker sessionpkg.AgeChecker) {
// 	s.mu.Lock()
// 	defer s.mu.Unlock()

// 	for {
// 		elem := s.list.Front()
// 		if elem == nil {
// 			break
// 		}
// 		bsi := elem.Value.(*basicSessionInfo)
// 		if !checker.ShouldReap(time.Unix(0, bsi.ct)) {
// 			break
// 		}
// 		if err := s.io.Delete(bsi.id); err == nil {
// 			s.list.Remove(elem)
// 			delete(s.m, bsi.id)
// 		}
// 	}
// }

// func (s *storage) update(sess *session) error {
// 	s.mu.Lock()
// 	defer s.mu.Unlock()

// 	if _, ok := s.m[sess.id]; ok {
// 		if err := s.io.Write(sess); err != nil {
// 			return err
// 		}
// 	}
// 	return nil
// }

// func (s *storage) setIO(io storageIO) {
// 	s.m = map[string]*list.Element{}
// 	s.list.Init()
// 	s.io = io
// }

// var _io = newStorageIO(filepath.Join(os.TempDir(), "gosessions"))
// var _storage = newStorage(_io)

// // Returns the storage.
// //
// // It's possible to set the path where the sessions files will
// // be created. To do this, just call this function giving a
// // valid string path. Keep in mind that all sessions will be lost.
// func Storage(args ...string) *storage {
// 	if len(args) == 0 {
// 		return _storage
// 	}
// 	path := args[0]
// 	if _, err := filepath.Abs(path); err != nil {
// 		panic("argument is not a valid path")
// 	}
// 	_storage.mu.Lock()
// 	defer _storage.mu.Unlock()

// 	_storage.setIO(newStorageIO(path))

// 	return _storage
// }

func init() {
	gob.Register(map[string]any{})
}
