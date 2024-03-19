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
	// Defines a value for the key.
	Set(key string, value any) error
	// Returns a value for the key, or nil if it doesn't exist.
	Get(key string) any
	// Removes the key and it's value.
	Delete(key string) error
	// Returns session identifier.
	SessionID() string
}

type Provider interface {
	SessionInit(sid string) (Session, error)
	SessionRead(sid string) (Session, error)
	SessionDestroy(sid string) error
	SessionGC(maxAge int64)
}

// Manager allows to work with sessions.
//
// It can start or destroy one session. Both operations will
// manipulate a http cookie.
//
// To start a session, it will:
// - Creates a new one, setting a http cookie; or
// - Retrieves, accordingly to the existent http cookie.
//
// To destroys a session, it can be forced or after it expires. The
// expired session is removed through the GC routine, which checks
// this condition.
type Manager struct {
	mu         sync.Mutex
	provider   Provider
	cookieName string
	maxAge     int64
}

// Returns a new Manager (address for pointer reference).
//
// The provider cannot be nil and cookie name cannot be empty.
func NewManager(provider Provider, cookieName string, maxAge int64) *Manager {
	if provider == nil {
		panic("nil provider")
	}
	if cookieName == "" {
		panic("empty cookie name")
	}
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

func (m *Manager) assertProviderAndCookieName() {
	if m.provider == nil {
		panic("nil provider")
	}
	if m.cookieName == "" {
		panic("empty cookie name")
	}
}

// Creates or retrieve the session based on the http cookie.
func (m *Manager) StartSession(w http.ResponseWriter, r *http.Request) (session Session) {
	m.assertProviderAndCookieName()
	m.mu.Lock()
	defer m.mu.Unlock()
	cookie, err := r.Cookie(m.cookieName)
	if err != nil || cookie.Value == "" {
		sid := m.sessionID()
		session, err = m.provider.SessionInit(sid)
		cookie := http.Cookie{Name: m.cookieName, Value: url.QueryEscape(sid), Path: "/", HttpOnly: true, MaxAge: int(m.maxAge)}
		http.SetCookie(w, &cookie)
	} else {
		sid, _ := url.QueryUnescape(cookie.Value)
		session, err = m.provider.SessionRead(sid)
	}
	if err != nil || session == nil {
		panic("unable to start the session")
	}
	return
}

// Destroys the session, finally cleaning up the http cookie.
func (m *Manager) DestroySession(w http.ResponseWriter, r *http.Request) {
	m.assertProviderAndCookieName()
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

// Creates a routine to check for expired sessions and remove them.
func (m *Manager) GC() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.provider.SessionGC(m.maxAge)
	time.AfterFunc(time.Duration(m.maxAge), func() {
		m.GC()
	})
}
