package session

import (
	"container/list"
	"testing"
	"time"

	"github.com/xandalm/go-session/testing/assert"
)

func TestSessionInit(t *testing.T) {

	dummyStorage := newStubStorage()

	t.Run("tell cache to add session", func(t *testing.T) {
		cache := &spyCache{}

		provider := &provider{
			cached:  cache,
			storage: dummyStorage,
		}

		_, err := provider.SessionInit("1")
		assert.Nil(t, err)

		if cache.callsToAdd == 0 {
			t.Error("didn't tell cache to add")
		}
	})

	cache := stubCache{}
	provider := &provider{
		cached:  cache,
		storage: dummyStorage,
	}

	t.Run("init the session", func(t *testing.T) {

		sid := "17af454"
		sess, err := provider.SessionInit(sid)

		assert.Nil(t, err)
		assert.NotNil(t, sess)

		if _, ok := cache[sid]; !ok {
			t.Error("didn't add session into cache")
		}
	})
	t.Run("returns error for empty sid", func(t *testing.T) {

		_, err := provider.SessionInit("")

		assert.Error(t, err, ErrEmptySessionId)
	})
	t.Run("returns error for duplicated sid", func(t *testing.T) {
		_, err := provider.SessionInit("17af454")

		assert.Error(t, err, ErrDuplicatedSessionId)
	})
}

func TestSessionRead(t *testing.T) {

	dummyStorage := newStubStorage()

	t.Run("tell cache to get session", func(t *testing.T) {
		cache := &spyCache{}

		provider := &provider{
			cached:  cache,
			storage: dummyStorage,
		}

		_, err := provider.SessionRead("1")
		assert.Nil(t, err)

		if cache.callsToGet == 0 {
			t.Error("didn't tell cache to get")
		}
	})

	cache := stubCache{
		"17af454": newStubSession("17af454"),
	}

	provider := &provider{
		cached:  cache,
		storage: dummyStorage,
	}

	t.Run("returns session", func(t *testing.T) {
		sid := "17af454"
		session, err := provider.SessionRead(sid)

		assert.Nil(t, err)
		assert.NotNil(t, session)

		if session.SessionID() != sid {
			t.Errorf("didn't get expected session, got %s but want %s", session.SessionID(), sid)
		}
	})
	t.Run("init session if has no session to read", func(t *testing.T) {
		sid := "17af450"
		session, err := provider.SessionRead(sid)

		assert.Nil(t, err)
		assert.NotNil(t, session)

		if session.SessionID() != sid {
			t.Fatalf("didn't get expected session, got %s but want %s", session.SessionID(), sid)
		}

		if _, ok := cache[sid]; !ok {
			t.Error("didn't add session into cache")
		}
	})
}

func TestSessionDestroy(t *testing.T) {

	session := newStubSession("17af454")
	registry := &stubStorageItem{
		session.SessionID(),
		map[string]any{},
	}
	storage := &stubStorage{
		data: map[string]StorageItem{
			registry.id: registry,
		},
	}
	cache := stubCache{
		session.SessionID(): session,
	}

	provider := &provider{
		cached:  cache,
		storage: storage,
	}

	t.Run("destroys session", func(t *testing.T) {
		sid := "17af454"
		err := provider.SessionDestroy(sid)

		assert.Nil(t, err)

		if _, ok := cache[sid]; ok {
			t.Fatalf("didn't remove session from cache")
		}
		if _, ok := storage.data[sid]; ok {
			t.Errorf("didn't remove session from storage")
		}
	})
}

func TestSessionGC(t *testing.T) {

	t.Run("destroy sessions that arrives max age", func(t *testing.T) {
		storage := &stubStorage{
			data: map[string]StorageItem{},
		}

		type cache struct {
			data map[string]Session
			mockCache
		}

		c := &cache{
			data: map[string]Session{},
		}
		c.AddFunc = func(s Session) {
			c.data[s.SessionID()] = s
		}
		c.ContainsFunc = func(s string) bool {
			_, ok := c.data[s]
			return ok
		}
		c.RemoveFunc = func(s string) {
			delete(c.data, s)
		}
		c.ExpiredSessionsFunc = func(ac AgeChecker) []string {
			ret := []string{}
			for k, v := range c.data {
				if ac.ShouldReap(v.(*session).ct) {
					delete(c.data, k)
					ret = append(ret, k)
				}
			}
			return ret
		}

		provider := &provider{
			cached:  c,
			storage: storage,
		}

		sid1 := "17af450"
		sid2 := "17af454"

		provider.SessionInit(sid1)

		time.Sleep(3 * time.Millisecond)

		provider.SessionInit(sid2)

		provider.SessionGC(stubMilliAgeChecker(1))

		if provider.cached.Contains(sid1) {
			t.Fatal("didn't destroy session")
		}

		if _, ok := provider.cached.(*cache).data[sid2]; !ok {
			t.Errorf("expected that the session %s is in the storage", sid2)
		}
	})
}

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
			time.Now(),
			time.Now(),
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
				time.Now(),
				time.Now(),
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
			time.Now(),
			time.Now(),
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
			time.Now(),
			time.Now(),
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
			time.Now(),
			time.Now(),
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
			time.Now(),
			time.Now(),
		},
	}

	time.Sleep(10 * time.Millisecond)

	node2 := &cacheNode{
		info: &sessionInfo{
			"2",
			time.Now(),
			time.Now(),
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
