package session

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type StubSession struct {
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
}

func (p *StubProvider) SessionInit(sid string) (Session, error) {
	sess := &StubSession{}
	return sess, nil
}

func (p *StubProvider) SessionRead(sid string) (Session, error) {
	return nil, nil
}

func (p *StubProvider) SessionDestroy(sid string) error {
	return nil
}

func (p *StubProvider) SessionGC(maxLifeTime int64) {}

func TestManager(t *testing.T) {
	cookieName := "SessionID"
	provider := &StubProvider{}
	manager := NewManager(provider, cookieName, 3600)

	asserNoNil(t, manager)

	t.Run("start a session", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "http://site.com", nil)
		res := httptest.NewRecorder()

		session := manager.StartSession(res, req)

		asserNoNil(t, session)

		cookie := getCookie(res)

		checkCookie(t, cookie, cookieName)

	})
}

func asserNoNil(t testing.TB, v any) {
	t.Helper()

	if v == nil {
		t.Fatal("expected no nil")
	}
}

func checkCookie(t *testing.T, cookie map[string]any, name string) {
	t.Helper()

	sid, ok := cookie[name]

	if !ok {
		t.Fatalf("didn't get %s cookie", name)
	}

	if sid.(string) == "" {
		t.Fatalf("got empty %s value", name)
	}
}

func getCookie(res *httptest.ResponseRecorder) (cookie map[string]any) {
	set_cookie := res.Header()["Set-Cookie"]

	cookie = make(map[string]any)

	if len(set_cookie) < 1 {
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
