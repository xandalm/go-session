package session

import "testing"

type StubProvider struct {
}

func (p *StubProvider) SessionInit(sid string) (Session, error) {
	return nil, nil
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
	got := NewManager(provider, cookieName, 3600)
	want := Manager{
		provider,
		cookieName,
		3600,
	}
	if *got != want {
		t.Errorf("got %v but want %v", got, want)
	}
}
