package session

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
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
	return ""
}

type StubProvider struct {
	sessions map[string]Session
}

func (p *StubProvider) SessionInit(sid string) (Session, error) {
	if p.sessions == nil {
		p.sessions = make(map[string]Session)
	}
	sess := &StubSession{
		id: sid,
	}
	p.sessions[sid] = sess
	return sess, nil
}

func (p *StubProvider) SessionRead(sid string) (Session, error) {
	sess := p.sessions[sid]
	return sess, nil
}

func (p *StubProvider) SessionDestroy(sid string) error {
	return nil
}

func (p *StubProvider) SessionGC(maxLifeTime int64) {}

var dummySite = "http://site.com"

func TestManager(t *testing.T) {
	cookieName := "SessionID"
	provider := &StubProvider{}
	manager := NewManager(provider, cookieName, 3600)

	assertNotNil(t, manager)

	t.Run("start a session", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, dummySite, nil)
		res := httptest.NewRecorder()

		session := manager.StartSession(res, req)
		assertNotNil(t, session)

		cookie := getCookie(res)
		assertNotNil(t, cookie)

		sid := cookie[cookieName]
		sidStr := sid.(string)
		assertNotEmpty(t, sidStr)

		assertEqual(t, sidStr, url.QueryEscape(session.(*StubSession).id))

		t.Run("restores same session", func(t *testing.T) {

			path := cookie["Path"].(string)
			httpOnly := cookie["HttpOnly"].(bool)
			maxAge, _ := strconv.Atoi(cookie["Max-Age"].(string))

			req, _ := http.NewRequest(http.MethodGet, dummySite+"/admin", nil)
			req.AddCookie(&http.Cookie{
				Name:     cookieName,
				Value:    sidStr,
				Path:     path,
				HttpOnly: httpOnly,
				MaxAge:   maxAge,
			})

			res := httptest.NewRecorder()

			session := manager.StartSession(res, req)
			assertNotNil(t, session)

			assertEqual(t, sidStr, url.QueryEscape(session.(*StubSession).id))
		})
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

func getCookie(res *httptest.ResponseRecorder) (cookie map[string]any) {
	set_cookie := res.Header()["Set-Cookie"]

	cookie = make(map[string]any)

	if len(set_cookie) != 1 {
		return nil
	}

	for _, pair := range strings.Split(set_cookie[0], "; ") {
		kv := strings.Split(pair, "=")
		if len(kv) > 1 {
			cookie[kv[0]] = kv[1]
			continue
		}
		cookie[kv[0]] = true
	}

	return
}
