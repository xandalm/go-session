package session_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/xandalm/go-session"
)

type StubSession struct {
	id string
}

func (s *StubSession) Set(key, value any) error {
	return nil
}

func (s *StubSession) Get(key any) any {
	return nil
}

func (s *StubSession) Delete(key any) error {
	return nil
}

func (s *StubSession) SessionID() string {
	return s.id
}

type StubProvider struct {
	sessions map[string]session.ISession
	doomed   []session.ISession
}

func (p *StubProvider) SessionInit(sid string) (session.ISession, error) {
	if p.sessions == nil {
		p.sessions = make(map[string]session.ISession)
	}
	sess := &StubSession{
		id: sid,
	}
	p.sessions[sid] = sess
	return sess, nil
}

func (p *StubProvider) SessionRead(sid string) (session.ISession, error) {
	sess := p.sessions[sid]
	return sess, nil
}

func (p *StubProvider) SessionDestroy(sid string) error {
	p.doomed = append(p.doomed, p.sessions[sid])
	return nil
}

func (p *StubProvider) SessionGC(maxLifeTime int64) {}

var dummySite = "http://site.com"

func TestManager(t *testing.T) {
	cookieName := "SessionID"
	provider := &StubProvider{}
	manager := session.NewManager(provider, cookieName, 3600)

	assertNotNil(t, manager)

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
		req, _ := http.NewRequest(http.MethodGet, dummySite, nil)
		res := httptest.NewRecorder()

		session := manager.StartSession(res, req)
		assertNotNil(t, session)

		cookie = parseCookie(getCookieFromResponse(res))
		assertNotNil(t, cookie)
		cookie.Name = cookieName

		sid := cookie.Value

		assertNotEmpty(t, sid)

		assertEqual(t, sid, url.QueryEscape(session.SessionID()))
	})

	t.Run("restores the same session", func(t *testing.T) {

		req, _ := http.NewRequest(http.MethodGet, dummySite+"/admin", nil)
		req.AddCookie(cookie)

		res := httptest.NewRecorder()

		session := manager.StartSession(res, req)
		assertNotNil(t, session)

		assertEqual(t, cookie.Value, url.QueryEscape(session.SessionID()))
	})

	t.Run("destroy the session", func(t *testing.T) {

		req, _ := http.NewRequest(http.MethodGet, dummySite, nil)
		req.AddCookie(cookie)

		res := httptest.NewRecorder()

		manager.DestroySession(res, req)

		sid, _ := url.QueryUnescape(cookie.Value)
		assertSessionIsDoomed(t, provider, sid)

		newCookie := parseCookie(getCookieFromResponse(res))
		assertNotNil(t, newCookie)

		if newCookie.Expires.After(time.Now()) || newCookie.MaxAge != 0 {
			t.Errorf("the cookie is not expired, Expires = %s and MaxAge = %d", newCookie.Expires, newCookie.MaxAge)
		}
	})
}

func assertNotNil(t testing.TB, v any) {
	t.Helper()

	if v == nil {
		t.Fatal("expected not nil")
	}
}

func assertNotEmpty(t testing.TB, v string) {
	t.Helper()

	if v == "" {
		t.Fatalf("expected not empty")
	}
}

func assertEqual(t testing.TB, a, b string) {
	t.Helper()

	if a != b {
		t.Fatalf("expected same values, but got %v and %v", a, b)
	}
}

func assertSessionIsDoomed(t testing.TB, provider *StubProvider, sid string) {
	t.Helper()

	doomed := false
	for _, d := range provider.doomed {
		if d.SessionID() == sid {
			doomed = true
			break
		}
	}

	if !doomed {
		t.Fatal("didn't set session to be removed from provider")
	}
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
