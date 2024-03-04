package filesystem

import (
	"os"
	"path/filepath"
	"time"

	sessionpkg "github.com/xandalm/go-session"
)

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
}

func NewStorage(path string, dir string) *storage {
	path, err := filepath.Abs(path)
	if err == nil {
		path = filepath.Join(path, dir)
		err = os.MkdirAll(path, 0750)
		if err == nil || os.IsExist(err) {
			return &storage{path}
		}
	}
	panic("session: cannot make sessions storage folder")
}

func (s *storage) CreateSession(sid string) (sessionpkg.Session, error) {
	filePath := filepath.Join(s.path, sid+".sess")
	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, 0666)
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
