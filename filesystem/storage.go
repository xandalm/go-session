package filesystem

import (
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/xandalm/go-session"
)

type Values = session.Values

type storage struct {
	mu     sync.Mutex
	path   string
	prefix string
}

func NewStorage(path, prefix string) *storage {
	err := os.MkdirAll(path, 0750)
	if err != nil && !os.IsExist(err) {
		panic(fmt.Sprintf("cannot make sessions storage folder, %v", err))
	}
	s := &storage{
		mu:     sync.Mutex{},
		path:   path,
		prefix: prefix,
	}
	return s
}

func (s *storage) Save(id string, values Values) error {
	s.mu.Lock()
	defer s.mu.Unlock()
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

func (s *storage) List() ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ret := []string{}
	entries, _ := os.ReadDir(s.path)
	for _, e := range entries {
		name := e.Name()
		if !strings.HasPrefix(name, s.prefix) {
			continue
		}
		ret = append(ret, name)
	}
	return ret, nil
}

func (s *storage) Read(id string) (Values, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data := make(Values)
	file, err := os.Open(filepath.Join(s.path, s.prefix+id))
	if err != nil {
		return data, err
	}
	defer file.Close()

	dec := gob.NewDecoder(file)
	if err := dec.Decode(&data); err != nil {
		return data, err
	}
	return data, nil
}

func (s *storage) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	err := os.Remove(filepath.Join(s.path, s.prefix+id))
	if err != nil {
		return err
	}
	return nil
}

func init() {
	gob.Register(Values{})
}
