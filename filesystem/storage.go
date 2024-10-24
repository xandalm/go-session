package filesystem

import (
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/xandalm/go-session"
)

type Values = session.Values

type sessFile struct {
	id string
	f  *os.File
}

type storage struct {
	mu     sync.Mutex
	path   string
	prefix string
	f      *os.File
	sfs    []*sessFile
}

func NewStorage(path, prefix string) *storage {
	err := os.MkdirAll(path, 0750)
	if err != nil && !os.IsExist(err) {
		panic(fmt.Sprintf("cannot create storage folder, %v", err))
	}
	f, err := os.Open(path)
	if err != nil {
		panic(fmt.Sprintf("unable to open storage folder, %v", err))
	}
	s := &storage{
		mu:     sync.Mutex{},
		path:   path,
		prefix: prefix,
		f:      f,
		sfs:    []*sessFile{},
	}
	return s
}

func (s *storage) getSessionFilePositon(id string) (int, bool) {
	return slices.BinarySearchFunc(s.sfs, id, func(sf *sessFile, id string) int {
		return strings.Compare(sf.id, id)
	})
}

func (s *storage) getFile(id string) (*os.File, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	pos, has := s.getSessionFilePositon(id)
	if has {
		f := s.sfs[pos].f
		f.Seek(0, 0)
		return f, nil
	}
	f, err := os.OpenFile(filepath.Join(s.path, s.prefix+id), os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}
	s.sfs = slices.Insert(s.sfs, pos, &sessFile{id, f})
	return f, nil
}

func (s *storage) Save(id string, values Values) error {
	file, err := s.getFile(id)
	if err != nil {
		return err
	}

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
	for _, sf := range s.sfs {
		ret = append(ret, sf.id)
	}
	return ret, nil
}

func (s *storage) Read(id string) (Values, error) {
	data := make(Values)
	file, err := s.getFile(id)
	if err != nil {
		return data, err
	}

	dec := gob.NewDecoder(file)
	if err := dec.Decode(&data); err != nil {
		return data, err
	}
	return data, nil
}

func (s *storage) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	pos, has := s.getSessionFilePositon(id)
	if !has {
		return nil
	}
	file := s.sfs[pos].f
	if err := file.Close(); err != nil {
		return err
	}
	err := os.Remove(filepath.Join(s.path, s.prefix+id))
	if err != nil {
		return err
	}
	return nil
}
