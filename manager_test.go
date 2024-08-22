package session

import (
	"context"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/xandalm/go-session/testing/assert"
)

var dummySite = "http://site.com"

func TestManager(t *testing.T) {
	cookieName := "SessionID"
	provider := &stubProvider{}
	dummyAdapter := stubMilliAgeCheckerAdapter
	manager := newManager(provider, cookieName, 3600, dummyAdapter)

	assert.NotNil(t, manager)

	var cookie *http.Cookie = nil

	parseCookie := func(cookie map[string]string) *http.Cookie {
		maxAge, _ := strconv.Atoi(cookie["Max-Age"])
		httpOnly, _ := strconv.ParseBool(cookie["HttpOnly"])
		c := &http.Cookie{
			Name:     cookieName,
			Value:    cookie[cookieName],
			Path:     cookie["Path"],
			HttpOnly: httpOnly,
			MaxAge:   maxAge,
		}
		expires, hasExpires := cookie["Expires"]
		if hasExpires {
			c.Expires, _ = time.Parse(time.RFC1123, expires)
		}
		return c
	}

	t.Run("start the session", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, dummySite, nil)
		res := httptest.NewRecorder()

		session := manager.StartSession(res, req)
		cancel()
		assert.NotNil(t, session)

		cookie = parseCookie(getCookieFromResponse(res))
		assert.NotNil(t, cookie)
		cookie.Name = cookieName

		sid := cookie.Value

		assert.NotEmpty(t, sid)

		assert.Equal(t, sid, url.QueryEscape(session.SessionID()))
	})

	t.Run("restores the same session", func(t *testing.T) {

		req, _ := http.NewRequest(http.MethodGet, dummySite+"/admin", nil)
		req.AddCookie(cookie)

		res := httptest.NewRecorder()

		session := manager.StartSession(res, req)
		assert.NotNil(t, session)

		assert.Equal(t, cookie.Value, url.QueryEscape(session.SessionID()))
	})

	t.Run("destroy the session", func(t *testing.T) {

		req, _ := http.NewRequest(http.MethodGet, dummySite, nil)
		req.AddCookie(cookie)

		res := httptest.NewRecorder()

		manager.DestroySession(res, req)

		sid, _ := url.QueryUnescape(cookie.Value)

		if _, ok := provider.Sessions[sid]; ok {
			t.Fatalf("didn't destroy session")
		}

		newCookie := parseCookie(getCookieFromResponse(res))
		assert.NotNil(t, newCookie)

		if newCookie.Expires.After(time.Now()) || newCookie.MaxAge != 0 {
			t.Errorf("the cookie is not expired, Expires = %s and MaxAge = %d", newCookie.Expires, newCookie.MaxAge)
		}
	})

	t.Run("panic when fail to start session", func(t *testing.T) {
		provider := &stubFailingProvider{}
		dummyAdapter := stubMilliAgeCheckerAdapter
		manager := newManager(provider, cookieName, 3600, dummyAdapter)

		defer func() {
			r := recover()
			if r != "session: unable to start the session" {
				t.Errorf("didn't get expected panic, got: %v", r)
			}
		}()

		req, _ := http.NewRequest(http.MethodGet, dummySite, nil)
		res := httptest.NewRecorder()

		manager.StartSession(res, req)
	})
}

func getCookieFromResponse(res *httptest.ResponseRecorder) (cookie map[string]string) {
	set_cookie := res.Header()["Set-Cookie"]

	cookie = make(map[string]string)

	if len(set_cookie) != 1 {
		return nil
	}

	for _, pair := range strings.Split(set_cookie[0], "; ") {
		kv := strings.Split(pair, "=")
		if len(kv) > 1 {
			cookie[kv[0]] = kv[1]
			continue
		}
		cookie[kv[0]] = "true"
	}

	return
}

func TestReset(t *testing.T) {
	cookieName := "SESSION"
	maxAge := int64(60)
	adapter := SecondsAgeCheckerAdapter
	storage := newStubStorage()
	sessionFactory := &mockSessionFactory{
		CreateFunc: func(s string) Session {
			return &stubSession{
				Id: s,
				V:  make(map[string]any),
			}
		},
		RestoreFunc: func(s string, m map[string]any) Session {
			return &stubSession{
				Id: s,
				V:  maps.Clone(m),
			}
		},
		OverrideValuesFunc: func(s Session, m map[string]any) {
			sess := s.(*stubSession)
			for k, v := range m {
				sess.V[k] = v
			}
		},
		ExtractValuesFunc: func(s Session) map[string]any {
			sess := s.(*stubSession)
			return maps.Clone(sess.V)
		},
	}
	Config(cookieName, maxAge, adapter, sessionFactory, storage)

	assert.NotNil(t, manager)
}
