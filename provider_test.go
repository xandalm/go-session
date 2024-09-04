package session

import (
	"container/list"
	"context"
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
		info := &sessionInfo{
			sess: sess,
			id:   sess.Id,
			ct:   sess.CreatedAt,
		}
		c.Add(info)

		if c.collec.Len() < 1 {
			t.Fatal("didn't add session on cache collection")
		}

		got := c.collec.Front().Value.(*cacheNode).info

		assert.Equal(t, got, info)

		t.Run("add in sid sorted index", func(t *testing.T) {
			if len(c.idx) < 1 {
				t.Fatal("didn't add session on sid based index")
			}

			node := c.idx[len(c.idx)-1]

			assert.NotNil(t, node)

			got := node.info

			assert.Equal(t, got, info)
		})

	})

	t.Run("keep sid index sorted", func(t *testing.T) {

		sess := &stubSession{
			Id:        "3",
			CreatedAt: NowTimeNanoseconds(),
		}
		node := &cacheNode{
			info: &sessionInfo{
				sess: sess,
				id:   sess.Id,
				ct:   sess.CreatedAt,
			},
		}

		collec := list.New()
		collec.PushBack(node)

		c := &cache{
			collec,
			[]*cacheNode{node},
		}

		sess = &stubSession{
			Id:        "1",
			CreatedAt: NowTimeNanoseconds(),
		}

		info := &sessionInfo{
			sess: sess,
			id:   sess.Id,
			ct:   sess.CreatedAt,
		}
		c.Add(info)

		if c.collec.Len() != 2 {
			t.Fatal("didn't add session")
		}

		if len(c.idx) != 2 {
			t.Fatal("didn't add session in sid sorted index")
		}

		if c.idx[0].info.id != "1" {
			t.Errorf("sid sorted index isn't sorted")
		}
	})
}

func TestCache_Contains(t *testing.T) {

	sess := &stubSession{
		Id:        "1",
		CreatedAt: NowTimeNanoseconds(),
	}
	node := &cacheNode{
		info: &sessionInfo{
			sess: sess,
			id:   sess.Id,
			ct:   sess.CreatedAt,
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

	sess := &stubSession{
		Id:        "1",
		CreatedAt: NowTimeNanoseconds(),
	}
	node := &cacheNode{
		info: &sessionInfo{
			sess: sess,
			id:   sess.Id,
			ct:   sess.CreatedAt,
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

	sess1 := &stubSession{
		Id:        "1",
		V:         Values{},
		CreatedAt: NowTimeNanoseconds(),
	}
	node1 := &cacheNode{
		info: &sessionInfo{
			sess: sess1,
			id:   sess1.Id,
			ct:   sess1.CreatedAt,
		},
	}

	time.Sleep(10 * time.Millisecond)

	sess2 := &stubSession{
		Id:        "2",
		V:         Values{},
		CreatedAt: NowTimeNanoseconds(),
	}
	node2 := &cacheNode{
		info: &sessionInfo{
			sess: sess2,
			id:   sess2.Id,
			ct:   sess2.CreatedAt,
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

type stubContext struct{}

func (s *stubContext) Deadline() (deadline time.Time, ok bool) {
	return time.Now(), false
}

func (s *stubContext) Done() <-chan struct{} {
	return make(<-chan struct{})
}

func (s *stubContext) Err() error {
	return nil
}

func (s *stubContext) Value(key any) any {
	return nil
}

func TestProvider_SessionInit(t *testing.T) {

	dummyStorage := newStubStorage()
	dummyContext := &stubContext{}

	sf := &mockSessionFactory{
		CreateFunc: func(id string, m Values, fn OnSessionMutation) Session {
			s := &stubSession{
				Id: id,
				V:  make(Values),
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
		sess, err := provider.SessionInit(dummyContext, sid)

		assert.NoError(t, err)
		assert.NotNil(t, sess)

		assert.NotNil(t, cache.Get(sid), "didn't add session into cache")
	})
	t.Run("returns error for empty sid", func(t *testing.T) {

		_, err := provider.SessionInit(dummyContext, "")

		assert.Equal(t, err, ErrEmptySessionId)
	})
	t.Run("returns error for duplicated sid", func(t *testing.T) {
		_, err := provider.SessionInit(dummyContext, "17af454")

		assert.Equal(t, err, ErrDuplicatedSessionId)
	})
}

func TestProvider_SessionRead(t *testing.T) {

	dummyStorage := newStubStorage()
	dummyContext := &stubContext{}

	cache := &cache{
		list.New(),
		[]*cacheNode{},
	}

	sf := &mockSessionFactory{
		CreateFunc: func(id string, m Values, fn OnSessionMutation) Session {
			s := &stubSession{
				Id: id,
				V:  make(Values),
			}
			maps.Copy(s.V, m)
			return s
		},
		RestoreFunc: func(id string, m Values, v Values, fn OnSessionMutation) Session {
			s := &stubSession{
				Id: id,
				V:  make(Values),
			}
			maps.Copy(s.V, m)
			maps.Copy(s.V, v)
			return s
		},
	}

	provider := &provider{
		ac: stubMilliAgeChecker(3600),
		ca: cache,
		st: dummyStorage,
		sf: sf,
	}

	sess := &stubSession{
		Id:        "17af454",
		CreatedAt: NowTimeNanoseconds(),
	}
	info := &sessionInfo{
		sess: sess,
		id:   sess.Id,
		ct:   sess.CreatedAt,
	}

	cache.Add(info)

	t.Run("returns session", func(t *testing.T) {
		sid := "17af454"
		got, err := provider.SessionRead(dummyContext, sid)

		assert.NoError(t, err)
		assert.NotNil(t, got)

		assert.Equal(t, got.(*stubSession), sess, "didn't get expected session, got %#v but want %#v", got.(*stubSession), sess)
	})
}

func TestProvider_SessionDestroy(t *testing.T) {

	provider := &provider{}

	storage := &stubStorage{
		data: map[string]Values{
			"17af454": {},
		},
	}

	cache := &cache{
		list.New(),
		[]*cacheNode{},
	}

	sid := "17af454"

	cache.Add(&sessionInfo{
		sess: &stubSession{
			Id: sid,
		},
		id: sid,
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

func TestProvider_SessionGC(t *testing.T) {

	t.Run("destroy sessions that arrives max age", func(t *testing.T) {
		storage := &stubStorage{
			data: map[string]Values{},
		}

		cache := &cache{
			list.New(),
			[]*cacheNode{},
		}

		provider := &provider{
			ac: stubMilliAgeChecker(1),
			ca: cache,
			st: storage,
		}

		sid1 := "17af450"
		sid2 := "17af454"

		now := time.Now().UnixNano()

		sess1 := &stubSession{
			Id:        sid1,
			V:         Values{"ct": now - int64(3*time.Millisecond)},
			CreatedAt: now - int64(3*time.Millisecond),
		}
		cache.Add(&sessionInfo{
			sess: sess1,
			id:   sess1.Id,
			ct:   sess1.CreatedAt,
		})
		storage.data[sid1] = Values{}

		sess2 := &stubSession{
			Id:        sid2,
			V:         Values{"ct": now},
			CreatedAt: now,
		}
		cache.Add(&sessionInfo{
			sess: sess2,
			id:   sess2.Id,
			ct:   sess2.CreatedAt,
		})
		storage.data[sid2] = Values{}

		provider.SessionGC()

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

func TestProvider_registerSessionPush(t *testing.T) {

	provider := &provider{}

	sid := "17af454"

	storage := &stubStorage{
		data: map[string]Values{
			sid: {},
		},
	}

	cache := &cache{
		list.New(),
		[]*cacheNode{},
	}

	factory := &mockSessionFactory{
		ExportValuesFunc: func(s Session) Values {
			return maps.Clone(s.(*stubSession).V)
		},
	}

	sess := &stubSession{
		Id:        sid,
		V:         Values{},
		CreatedAt: NowTimeNanoseconds(),
	}

	info := &sessionInfo{
		sess: sess,
		id:   sess.Id,
		ct:   sess.CreatedAt,
	}
	cache.Add(info)

	provider.ca = cache
	provider.st = storage
	provider.sf = factory

	sess.V["foo"] = "bar"

	info.watchers++
	info.mu.Lock()

	ctx, cancel := context.WithCancel(context.Background())
	provider.registerSessionPush(ctx, info)

	cancel()

	time.Sleep(1 * time.Microsecond)

	assert.Equal(t, storage.data[sid], sess.V)
	assert.Nil(t, info.sess)
}
