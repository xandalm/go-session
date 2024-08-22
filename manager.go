package session

import (
	"crypto/rand"
	"encoding/base64"
	"io"
	"net/http"
	"net/url"
	"time"
)

type Session interface {
	SessionID() string
	Set(string, any) error
	Get(string) any
	Delete(string) error
}

type SessionFactory interface {
	// Creates a Session with the given id and the meta values.
	// It's suggested that the meta values can't be manipulated
	// through [Session.Set] or [Session.Get]
	Create(id string, m map[string]any) Session
	// Similar to Create method, but this method assume that the
	// session is being restored, allowing to put defined values.
	Restore(id string, m map[string]any, v map[string]any) Session
	// Merge session values, overwriting old values and
	// adding coming new values. The not collided values must
	// be kept. Both common and meta values can be changed by
	// this method.
	OverrideValues(sess Session, v map[string]any)
	// Return all values, including common and meta values.
	ExtractValues(sess Session) map[string]any
}

// type StorageItem interface {
// 	Id() string
// 	Set(k string, v any)
// 	Delete(k string)
// 	Values() map[string]any
// }

type Storage interface {
	Save(string, map[string]any) error
	Read(string) (map[string]any, error)
	List() ([]string, error)
	Delete(string) error
}

type Provider interface {
	SessionInit(sid string) (Session, error)
	SessionRead(sid string) (Session, error)
	SessionDestroy(sid string) error
	SessionPush(Session) error
	SessionPull(Session) error
	SessionGC(checker AgeChecker)
}

type AgeChecker interface {
	// It says if the session is expired.
	// The given value is the unix nanoseconds
	// count until the session creation time.
	ShouldReap(int64) bool
}

type AgeCheckerAdapter func(int64) AgeChecker

type secondsAgeChecker int64

func (ma secondsAgeChecker) ShouldReap(t int64) bool {
	diff := time.Now().Unix() - (t / int64(time.Second))
	return diff >= int64(ma)
}

var SecondsAgeCheckerAdapter AgeCheckerAdapter = func(maxAge int64) AgeChecker {
	return secondsAgeChecker(maxAge)
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
	provider   Provider
	cookieName string
	maxAge     int64
	adapter    AgeCheckerAdapter
	timer      *time.Timer
}

// Returns a new Manager (address for pointer reference).
//
// The provider cannot be nil and cookie name cannot be empty.
func newManager(provider Provider, cookieName string, maxAge int64, adapter AgeCheckerAdapter) *Manager {
	if provider == nil {
		panic("session: nil provider")
	}
	if cookieName == "" {
		panic("session: empty cookie name")
	}
	return &Manager{
		provider:   provider,
		cookieName: cookieName,
		maxAge:     maxAge,
		adapter:    adapter,
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
		panic("session: nil provider")
	}
	if m.cookieName == "" {
		panic("session: empty cookie name")
	}
}

// Creates or retrieve the session based on the http cookie.
func (m *Manager) StartSession(w http.ResponseWriter, r *http.Request) (session Session) {
	m.assertProviderAndCookieName()
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
		panic("session: unable to start the session")
	}
	// go func(ctx context.Context) {
	// 	<-ctx.Done()
	// 	m.provider.SessionSync(session)
	// }(r.Context())
	return
}

// Destroys the session, finally cleaning up the http cookie.
func (m *Manager) DestroySession(w http.ResponseWriter, r *http.Request) {
	m.assertProviderAndCookieName()
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
	m.provider.SessionGC(m.adapter(m.maxAge))
	m.timer = time.AfterFunc(time.Duration(m.maxAge), func() {
		m.GC()
	})
}

var manager *Manager

// Configure the session cookie name, the session expiration
// time and where the sessions values will be held (storage).
func Config(cookieName string, maxAge int64, adapter AgeCheckerAdapter, sessionFactory SessionFactory, storage Storage) {
	if manager != nil && manager.timer != nil {
		manager.timer.Stop()
	}
	manager = newManager(
		newProvider(sessionFactory, storage),
		cookieName,
		maxAge,
		adapter,
	)
	manager.GC()
}

func assertIsConfigured() {
	if manager == nil {
		panic("session: must configure session manager, use Config method")
	}
}

// Starts the session.
//
// Creates a new one, or restores accordingly to HTTP cookie.
func Start(w http.ResponseWriter, r *http.Request) Session {
	assertIsConfigured()
	return manager.StartSession(w, r)
}

// Destroys the session.
//
// The HTTP cookie will be removed.
func Destroy(w http.ResponseWriter, r *http.Request) {
	assertIsConfigured()
	manager.DestroySession(w, r)
}
