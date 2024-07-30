package session

import (
	"container/list"
	"testing"
	"time"

	"github.com/xandalm/go-session/testing/assert"
)

var dummyAdapter = func(maxAge int64) AgeChecker {
	return nil
}

func TestSessionInit(t *testing.T) {

	dummyStorage := newStubSessionStorage()

	t.Run("tell cache to add session", func(t *testing.T) {
		cache := &spyCache{}

		provider := &defaultProvider{cache, dummyStorage, dummyAdapter}

		_, err := provider.SessionInit("1")
		assert.Nil(t, err)

		if cache.callsToAdd == 0 {
			t.Error("didn't tell cache to add")
		}
	})

	provider := &defaultProvider{newCache(), dummyStorage, dummyAdapter}
	t.Run("init the session", func(t *testing.T) {

		sid := "17af454"
		sess, err := provider.SessionInit(sid)

		assert.Nil(t, err)
		assert.NotNil(t, sess)
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

	dummyStorage := newStubSessionStorage()

	t.Run("tell cache to get session", func(t *testing.T) {
		cache := &spyCache{}

		provider := &defaultProvider{cache, dummyStorage, dummyAdapter}

		_, err := provider.SessionRead("1")
		assert.Nil(t, err)

		if cache.callsToGet == 0 {
			t.Error("didn't tell cache to get")
		}
	})

	cache := stubCache{
		"17af454": newStubSession("17af454"),
	}

	provider := &defaultProvider{cache, dummyStorage, dummyAdapter}

	t.Run("returns session", func(t *testing.T) {
		sid := "17af454"
		session, err := provider.SessionRead(sid)

		assert.Nil(t, err)
		assert.NotNil(t, session)

		if session.SessionID() != sid {
			t.Errorf("didn't get expected session, got %s but want %s", session.SessionID(), sid)
		}
	})
	t.Run("start new session if has no session to read", func(t *testing.T) {
		sid := "17af450"
		session, err := provider.SessionRead(sid)

		assert.Nil(t, err)
		assert.NotNil(t, session)

		if session.SessionID() != sid {
			t.Errorf("didn't get expected session, got %s but want %s", session.SessionID(), sid)
		}
	})
}

func TestSessionDestroy(t *testing.T) {

	sessionStorage := &stubSessionStorage{
		Sessions: map[string]Session{
			"17af454": newStubSession("17af454"),
		},
	}

	provider := &defaultProvider{newCache(), sessionStorage, dummyAdapter}

	t.Run("destroys session", func(t *testing.T) {
		sid := "17af454"
		err := provider.SessionDestroy(sid)

		assert.Nil(t, err)

		if _, ok := sessionStorage.Sessions[sid]; ok {
			t.Fatalf("didn't destroy session")
		}
	})
	t.Run("returns error for destroy failing", func(t *testing.T) {
		sessionStorage := &stubFailingSessionStorage{}
		provider := &defaultProvider{newCache(), sessionStorage, dummyAdapter}

		err := provider.SessionDestroy("17af454")

		assert.Error(t, err, ErrUnableToDestroySession)
	})
}

func TestSessionGC(t *testing.T) {

	t.Run("destroy sessions that arrives max age", func(t *testing.T) {
		sessionStorage := &stubSessionStorage{
			Sessions: map[string]Session{},
		}

		provider := &defaultProvider{newCache(), sessionStorage, func(maxAge int64) AgeChecker {
			return stubMilliAgeChecker(maxAge)
		}}

		sid1 := "17af450"
		sid2 := "17af454"

		provider.SessionInit(sid1)

		time.Sleep(3 * time.Millisecond)

		provider.SessionInit(sid2)

		provider.SessionGC(1)

		if provider.cached.Contains(sid1) {
			t.Fatal("didn't destroy session")
		}

		if provider.cached.(*cache).collec.Len() != 1 {
			t.Errorf("expected the session with id=%s in storage", sid2)
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
