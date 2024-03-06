package filesystem

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	sessionpkg "github.com/xandalm/go-session"
)

type session struct {
	id string
	v  map[string]any
	ct time.Time
	at time.Time
}

func (s *session) MarshalBinary() ([]byte, error) {
	var b bytes.Buffer
	fmt.Fprintln(&b, s.ct.UnixNano(), s.at.UnixNano())
	return b.Bytes(), nil
}

func (s *session) UnmarshalBinary(data []byte) error {
	b := bytes.NewBuffer(data)
	var cts int64
	var ats int64
	_, err := fmt.Fscanln(b, &cts, &ats)
	s.ct = time.Unix(0, cts)
	s.at = time.Unix(0, ats)
	return err
}

func (s *session) SessionID() string {
	return s.id
}

func (s *session) Get(key string) any {
	return s.v[key]
}

func (s *session) Set(key string, value any) error {
	s.v[key] = value
	return nil
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
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, err
	}
	return &session{
		id: sid,
		v:  map[string]any{},
		ct: fileInfo.ModTime(),
	}, nil
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

	var sess session
	dec := gob.NewDecoder(r)

	err := dec.Decode(&sess)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return &sess, nil
}

func (s storage) filePath(sid string) string {
	return filepath.Join(s.path, sid+"."+s.ext)
}
