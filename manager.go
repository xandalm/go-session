package session

import (
	"crypto/rand"
	"encoding/base64"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

type Session interface {
	Set(key string, value any) error
	Get(key string) any
	Delete(key string) error
	Values() SessionValues
	SessionID() string
	CreationTime() time.Time
}

type Provider interface {
	SessionInit(sid string) (Session, error)
	SessionRead(sid string) (Session, error)
	SessionDestroy(sid string) error
	SessionGC(maxAge int64)
}

type Manager struct {
	mu         sync.Mutex
	provider   Provider
	cookieName string
	maxAge     int64
}

func NewManager(provider Provider, cookieName string, maxAge int64) *Manager {
	return &Manager{
		provider:   provider,
		cookieName: cookieName,
		maxAge:     maxAge,
	}
}

func (m *Manager) sessionID() string {
	b := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return ""
	}
	return base64.URLEncoding.EncodeToString(b)
}

func (m *Manager) StartSession(w http.ResponseWriter, r *http.Request) (session Session) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cookie, err := r.Cookie(m.cookieName)
	if err != nil || cookie.Value == "" {
		sid := m.sessionID()
		session, _ = m.provider.SessionInit(sid)
		cookie := http.Cookie{Name: m.cookieName, Value: url.QueryEscape(sid), Path: "/", HttpOnly: true, MaxAge: int(m.maxAge)}
		http.SetCookie(w, &cookie)
	} else {
		sid, _ := url.QueryUnescape(cookie.Value)
		session, _ = m.provider.SessionRead(sid)
	}
	return
}

func (m *Manager) DestroySession(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cookie, err := r.Cookie(m.cookieName)
	if err != nil || cookie.Value == "" {
		return
	} else {
		sid, _ := url.QueryUnescape(cookie.Value)
		m.provider.SessionDestroy(sid)
		cookie := http.Cookie{Name: m.cookieName, Value: url.QueryEscape(sid), Path: "/", HttpOnly: true, Expires: time.Now(), MaxAge: -1}
		http.SetCookie(w, &cookie)
	}
}

func (m *Manager) GC() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.provider.SessionGC(m.maxAge)
	time.AfterFunc(time.Duration(m.maxAge), func() {
		m.GC()
	})
}
