package session

import (
	"container/list"
	"maps"
	"testing"
	"time"

	"github.com/xandalm/go-session/testing/assert"
)

func TestCache_Add(t *testing.T) {
	t.Run("add session", func(t *testing.T) {
		c := &cache{
			list.New(),
			[]*cacheNode{},
		}

		sess := &stubSession{
			Id:        "1",
			CreatedAt: NowTimeNanoseconds(),
		}
		c.Add(sess)

		if c.collec.Len() < 1 {
			t.Fatal("didn't add session on cache collection")
		}

		got := c.collec.Front().Value.(*cacheNode).sess.(*stubSession)

		assert.Equal(t, got, sess)

		t.Run("add in sid sorted index", func(t *testing.T) {
			if len(c.idx) < 1 {
				t.Fatal("didn't add session on sid based index")
			}

			node := c.idx[len(c.idx)-1]

			assert.NotNil(t, node)

			got := node.sess.(*stubSession)

			assert.Equal(t, got, sess)
		})

	})

	t.Run("keep sid index sorted", func(t *testing.T) {

		node := &cacheNode{
			sess: &stubSession{
				Id:        "3",
				CreatedAt: NowTimeNanoseconds(),
			},
		}

		collec := list.New()
		collec.PushBack(node)

		c := &cache{
			collec,
			[]*cacheNode{node},
		}

		c.Add(&stubSession{
			Id:        "1",
			CreatedAt: NowTimeNanoseconds(),
		})

		if c.collec.Len() != 2 {
			t.Fatal("didn't add session")
		}

		if len(c.idx) != 2 {
			t.Fatal("didn't add session in sid sorted index")
		}

		if c.idx[0].sess.(*stubSession).Id != "1" {
			t.Errorf("sid sorted index isn't sorted")
		}
	})
}

func TestCache_Contains(t *testing.T) {

	node := &cacheNode{
		sess: &stubSession{
			Id:        "1",
			CreatedAt: NowTimeNanoseconds(),
		},
	}

	collec := list.New()
	collec.PushBack(node)

	c := &cache{
		collec,
		[]*cacheNode{node},
	}

	t.Run("returns true", func(t *testing.T) {

		got := c.Contains("1")

		if !got {
			t.Error("didn't get true")
		}
	})

	t.Run("returns false", func(t *testing.T) {

		got := c.Contains("2")

		if got {
			t.Error("didn't get false")
		}
	})
}

func TestCache_Remove(t *testing.T) {

	node := &cacheNode{
		sess: &stubSession{
			Id:        "1",
			CreatedAt: NowTimeNanoseconds(),
		},
	}

	collec := list.New()
	node.anchor = collec.PushBack(node)

	c := &cache{
		collec,
		[]*cacheNode{node},
	}

	c.Remove("1")

	if c.collec.Len() == 1 {
		t.Fatal("didn't remove from cache collection")
	}

	if len(c.idx) == 1 {
		t.Fatal("didn't remove from sid sorted index")
	}
}

func TestCache_ExpiredSessions(t *testing.T) {

	collec := list.New()

	node1 := &cacheNode{
		sess: &stubSession{
			Id:        "1",
			CreatedAt: NowTimeNanoseconds(),
		},
	}

	time.Sleep(10 * time.Millisecond)

	node2 := &cacheNode{
		sess: &stubSession{
			Id:        "2",
			V:         map[string]any{},
			CreatedAt: NowTimeNanoseconds(),
		},
	}

	node1.anchor = collec.PushBack(node1)

	node2.anchor = collec.PushBack(node2)

	c := &cache{
		collec,
		[]*cacheNode{node1, node2},
	}

	t.Run("remove expired session and return its sid", func(t *testing.T) {
		removed := c.ExpiredSessions(stubMilliAgeChecker(1))

		assert.NotNil(t, removed)

		if len(removed) < 1 {
			t.Fatal("didn't return sid")
		}

		if removed[0] != "1" {
			t.Fatal("didn't return \"1\" in removed sid list")
		}

		if c.Contains("1") {
			t.Fatal("didn't remove the first (expired) session")
		}

		if !c.Contains("2") {
			t.Error("second session should not be removed")
		}
	})
}

func TestProvider_SessionInit(t *testing.T) {

	dummyStorage := newStubStorage()

	sf := &mockSessionFactory{
		CreateFunc: func(id string, m map[string]any) Session {
			s := &stubSession{
				Id: id,
				V:  make(map[string]any),
			}
			maps.Copy(s.V, m)
			return s
		},
	}

	cache := &cache{
		list.New(),
		[]*cacheNode{},
	}
	provider := &provider{
		ca: cache,
		st: dummyStorage,
		sf: sf,
	}

	t.Run("init the session", func(t *testing.T) {

		sid := "17af454"
		sess, err := provider.SessionInit(sid)

		assert.NoError(t, err)
		assert.NotNil(t, sess)

		assert.NotNil(t, cache.Get(sid), "didn't add session into cache")
	})
	t.Run("returns error for empty sid", func(t *testing.T) {

		_, err := provider.SessionInit("")

		assert.Equal(t, err, ErrEmptySessionId)
	})
	t.Run("returns error for duplicated sid", func(t *testing.T) {
		_, err := provider.SessionInit("17af454")

		assert.Equal(t, err, ErrDuplicatedSessionId)
	})
}

func TestProvider_SessionRead(t *testing.T) {

	dummyStorage := newStubStorage()

	cache := &cache{
		list.New(),
		[]*cacheNode{},
	}

	sf := &mockSessionFactory{
		CreateFunc: func(id string, m map[string]any) Session {
			s := &stubSession{
				Id: id,
				V:  make(map[string]any),
			}
			maps.Copy(s.V, m)
			return s
		},
		RestoreFunc: func(id string, m map[string]any, v map[string]any) Session {
			s := &stubSession{
				Id: id,
				V:  make(map[string]any),
			}
			maps.Copy(s.V, m)
			maps.Copy(s.V, v)
			return s
		},
	}

	provider := &provider{
		ca: cache,
		st: dummyStorage,
		sf: sf,
	}

	sess := &stubSession{
		Id:        "17af454",
		CreatedAt: NowTimeNanoseconds(),
	}

	cache.Add(sess)

	t.Run("returns session", func(t *testing.T) {
		sid := "17af454"
		got, err := provider.SessionRead(sid)

		assert.NoError(t, err)
		assert.NotNil(t, got)

		assert.Equal(t, got.(*stubSession), sess, "didn't get expected session, got %#v but want %#v", got.(*stubSession), sess)
	})

	t.Run("init session if has no session to read", func(t *testing.T) {
		sid := "17af450"
		got, err := provider.SessionRead(sid)

		assert.NoError(t, err)
		assert.NotNil(t, got)

		sess := got.(*stubSession)
		if sess.Id != sid {
			t.Fatalf("didn't get expected session, got %s but want %s", sess.Id, sid)
		}

		assert.NotNil(t, cache.Get(sid), "didn't add session into cache")
	})
}

func TestProvider_SessionDestroy(t *testing.T) {

	provider := &provider{}

	storage := &stubStorage{
		data: map[string]map[string]any{
			"17af454": {},
		},
	}

	cache := &cache{
		list.New(),
		[]*cacheNode{},
	}

	sid := "17af454"

	cache.Add(&stubSession{
		Id: sid,
	})

	provider.ca = cache
	provider.st = storage

	t.Run("destroys session", func(t *testing.T) {
		err := provider.SessionDestroy(sid)

		assert.NoError(t, err)

		got := cache.Get(sid)

		assert.Nil(t, got, "didn't remove session from cache")

		if _, ok := storage.data[sid]; ok {
			t.Errorf("didn't remove session from storage")
		}
	})
}

func TestProvider_SessionSynchronization(t *testing.T) {

	sid := "17af454"
	svalues := map[string]any{
		"foo": "bar",
	}

	storage := &stubStorage{
		data: map[string]map[string]any{
			sid: svalues,
		},
	}
	dummyCache := &cache{
		list.New(),
		[]*cacheNode{},
	}

	sf := &mockSessionFactory{
		OverrideValuesFunc: func(s Session, m map[string]any) {
			sess := s.(*stubSession)
			values := maps.Clone(m)
			for k, v := range sess.V {
				values[k] = v
			}
			sess.V = values
		},
		ExtractValuesFunc: func(s Session) map[string]any {
			sess := s.(*stubSession)
			return maps.Clone(sess.V)
		},
	}

	provider := &provider{
		ca: dummyCache,
		st: storage,
		sf: sf,
	}

	t.Run("pull session data from storage", func(t *testing.T) {
		sess := &stubSession{
			Id: "17af454",
			V:  map[string]any{},
		}
		provider.SessionPull(sess)

		want := map[string]any{"foo": "bar"}

		assert.Equal(t, sess.V, want)
	})

	t.Run("push session data to storage too", func(t *testing.T) {
		sess := &stubSession{
			Id: "17af454",
			V:  map[string]any{},
		}
		sess.V["key"] = "value"
		provider.SessionPush(sess)

		got := storage.data[sess.Id]
		want := map[string]any{"foo": "bar", "key": "value"}

		assert.Equal(t, got, want)
	})
}

func TestProvider_SessionGC(t *testing.T) {

	t.Run("destroy sessions that arrives max age", func(t *testing.T) {
		storage := &stubStorage{
			data: map[string]map[string]any{},
		}

		cache := &cache{
			list.New(),
			[]*cacheNode{},
		}

		provider := &provider{
			ca: cache,
			st: storage,
		}

		sid1 := "17af450"
		sid2 := "17af454"

		now := time.Now().UnixNano()

		cache.Add(&stubSession{
			Id:        sid1,
			V:         map[string]any{"ct": now - int64(3*time.Millisecond)},
			CreatedAt: now - int64(3*time.Millisecond),
		})
		storage.data[sid1] = map[string]any{}

		cache.Add(&stubSession{
			Id:        sid2,
			V:         map[string]any{"ct": now},
			CreatedAt: now,
		})
		storage.data[sid2] = map[string]any{}

		provider.SessionGC(stubMilliAgeChecker(1))

		if cache.Contains(sid1) {
			t.Fatal("didn't destroy session from cache")
		}

		if _, ok := storage.data[sid1]; ok {
			t.Fatal("didn't destroy session from storage")
		}

		if !cache.Contains(sid2) {
			t.Fatalf("expected that the session %s is in the cache", sid2)
		}

		if _, ok := storage.data[sid2]; !ok {
			t.Errorf("expected that the session %s is in the storage", sid2)
		}
	})
}

func TestProvider(t *testing.T) {
	s := &stubStorage{
		data: map[string]map[string]any{
			"1": {"foo": "bar"},
		},
	}
	sf := &mockSessionFactory{
		CreateFunc: func(id string, m map[string]any) Session {
			s := &stubSession{
				Id: id,
				V:  map[string]any{},
			}
			maps.Copy(s.V, m)
			return s
		},
		RestoreFunc: func(id string, m map[string]any, v map[string]any) Session {
			s := &stubSession{
				Id: id,
				V:  map[string]any{},
			}
			maps.Copy(s.V, m)
			maps.Copy(s.V, v)
			return s
		},
	}

	p := newProvider(sf, s)

	assert.NotNil(t, p)

	p.ca.Contains("1")

	got1, err := p.SessionRead("1")
	assert.NoError(t, err)

	assert.Equal(t, got1.Get("foo"), "bar")

	// p.sessionInit("2")
}
