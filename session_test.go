package session

import (
	"net/http"
	"net/http/httptest"
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
	want := Manager{
		provider,
		cookieName,
		3600,
	}

	asserNoNil(t, manager)

	assertSameManager(t, *manager, want)

	t.Run("start a session", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "http://site.com", nil)
		res := httptest.NewRecorder()

		session := manager.StartSession(res, req)

		if session == nil {
			t.Error("didn't get session")
		}
	})
}

func asserNoNil(t testing.TB, v any) {
	t.Helper()

	if v == nil {
		t.Fatalf("expected no nil, got %v", v)
	}
}

func assertSameManager(t testing.TB, a, b Manager) {
	t.Helper()

	if a != b {
		t.Fatalf("expected %v to be equal %v", a, b)
	}
}
