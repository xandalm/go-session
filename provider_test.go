package session

import (
	"container/list"
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
		s := &session{
			nil,
			"1",
			map[string]any{},
			NowTimeNanoseconds(),
			NowTimeNanoseconds(),
		}

		c.Add(s)

		if c.collec.Len() < 1 {
			t.Fatal("didn't add session on cache collection")
		}

		got := c.collec.Front().Value.(*cacheNode).info
		want := &sessionInfo{
			s.id,
			s.ct,
			s.at,
		}

		assert.Equal(t, got, want)

		t.Run("add in sid sorted index", func(t *testing.T) {
			if len(c.sidIdx) < 1 {
				t.Fatal("didn't add session on sid based index")
			}

			node := c.sidIdx[len(c.sidIdx)-1]

			assert.NotNil(t, node)

			got := node.info

			assert.Equal(t, got, want)
		})

	})

	t.Run("keep sid index sorted", func(t *testing.T) {

		node := &cacheNode{
			info: &sessionInfo{
				"3",
				NowTimeNanoseconds(),
				NowTimeNanoseconds(),
			},
		}

		collec := list.New()
		collec.PushBack(node)

		c := &cache{
			collec,
			[]*cacheNode{node},
		}

		c.Add(&session{
			nil,
			"1",
			map[string]any{},
			NowTimeNanoseconds(),
			NowTimeNanoseconds(),
		})

		if c.collec.Len() != 2 {
			t.Fatal("didn't add session")
		}

		if len(c.sidIdx) != 2 {
			t.Fatal("didn't add session in sid sorted index")
		}

		if c.sidIdx[0].info.sid != "1" {
			t.Errorf("sid sorted index isn't sorted")
		}
	})
}

func TestCache_Contains(t *testing.T) {

	node := &cacheNode{
		info: &sessionInfo{
			"1",
			NowTimeNanoseconds(),
			NowTimeNanoseconds(),
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
		info: &sessionInfo{
			"1",
			NowTimeNanoseconds(),
			NowTimeNanoseconds(),
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

	if len(c.sidIdx) == 1 {
		t.Fatal("didn't remove from sid sorted index")
	}
}

func TestCache_ExpiredSessions(t *testing.T) {

	collec := list.New()

	node1 := &cacheNode{
		info: &sessionInfo{
			"1",
			NowTimeNanoseconds(),
			NowTimeNanoseconds(),
		},
	}

	time.Sleep(10 * time.Millisecond)

	node2 := &cacheNode{
		info: &sessionInfo{
			"2",
			NowTimeNanoseconds(),
			NowTimeNanoseconds(),
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

func TestSessionInit(t *testing.T) {

	dummyStorage := newStubStorage()

	cache := &cache{
		list.New(),
		[]*cacheNode{},
	}
	provider := &provider{
		cached:  cache,
		storage: dummyStorage,
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

func TestSessionRead(t *testing.T) {

	dummyStorage := newStubStorage()

	cache := &cache{
		list.New(),
		[]*cacheNode{},
	}

	provider := &provider{
		cached:  cache,
		storage: dummyStorage,
	}

	sess := &session{
		p:  provider,
		id: "17af454",
		v:  map[string]any{},
		ct: time.Now().UnixNano(),
		at: time.Now().UnixNano(),
	}

	cache.Add(sess)

	t.Run("returns session", func(t *testing.T) {
		sid := "17af454"
		got, err := provider.SessionRead(sid)

		assert.NoError(t, err)
		assert.NotNil(t, got)

		assert.Equal(t, *got.(*session), *sess, "didn't get expected session, got %#v but want %#v", *got.(*session), *sess)
	})

	t.Run("init session if has no session to read", func(t *testing.T) {
		sid := "17af450"
		got, err := provider.SessionRead(sid)

		assert.NoError(t, err)
		assert.NotNil(t, got)

		sess := got.(*session)
		if sess.id != sid {
			t.Fatalf("didn't get expected session, got %s but want %s", sess.id, sid)
		}

		assert.NotNil(t, cache.Get(sid), "didn't add session into cache")
	})
}

func TestSessionDestroy(t *testing.T) {

	provider := &provider{}

	sess := &session{
		p:  provider,
		id: "17af454",
		v:  map[string]any{},
	}
	storage := &stubStorage{
		data: map[string]map[string]any{
			"17af454": {},
		},
	}

	cache := &cache{
		list.New(),
		[]*cacheNode{},
	}
	cache.Add(sess)

	provider.cached = cache
	provider.storage = storage

	t.Run("destroys session", func(t *testing.T) {
		sid := "17af454"
		err := provider.SessionDestroy(sid)

		assert.NoError(t, err)

		got := cache.Get(sid)

		assert.Nil(t, got, "didn't remove session from cache")

		if _, ok := storage.data[sid]; ok {
			t.Errorf("didn't remove session from storage")
		}
	})
}

func TestSessionSync(t *testing.T) {

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

	provider := &provider{
		cached:  dummyCache,
		storage: storage,
	}

	t.Run("pull session data from storage", func(t *testing.T) {
		sess := &session{
			p:  provider,
			id: "17af454",
			v:  map[string]any{},
		}
		provider.SessionSync(sess)

		want := session{
			p:  provider,
			id: "17af454",
			v:  map[string]any{"foo": "bar"},
		}

		assert.Equal(t, *sess, want)
	})

	t.Run("push session data to storage too", func(t *testing.T) {
		sess := &session{
			p:  provider,
			id: "17af454",
			v:  map[string]any{},
		}
		sess.v["key"] = "value"
		provider.SessionSync(sess)

		got := storage.data[sess.id]
		want := sess.v

		assert.Equal(t, got, want)
	})
}

func TestSessionGC(t *testing.T) {

	t.Run("destroy sessions that arrives max age", func(t *testing.T) {
		storage := &stubStorage{
			data: map[string]map[string]any{},
		}

		cache := &cache{
			list.New(),
			[]*cacheNode{},
		}

		provider := &provider{
			cached:  cache,
			storage: storage,
		}

		sid1 := "17af450"
		sid2 := "17af454"

		now := time.Now().UnixNano()

		cache.Add(&session{
			provider,
			sid1,
			map[string]any{},
			now - int64(3*time.Millisecond),
			now - int64(3*time.Millisecond),
		})
		storage.data[sid1] = map[string]any{}

		cache.Add(&session{
			provider,
			sid2,
			map[string]any{},
			now,
			now,
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
