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
	SessionID() string
	Set(string, any) error
	Get(string) any
	Delete(string) error
	values() map[string]any
}

type StorageItem interface {
	Id() string
	Set(k string, v any)
	Delete(k string)
	Values() map[string]any
}

type Storage interface {
	Save(Session, Session2StorageItem) error
	Load(id string, adapter StorageItem2Session) (Session, error)
	Delete(id string) error
}

type Session2StorageItem func(Session) StorageItem
type StorageItem2Session func(StorageItem) Session

type Provider interface {
	SessionInit(sid string) (Session, error)
	SessionRead(sid string) (Session, error)
	SessionDestroy(sid string) error
	SessionSync(Session) error
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
	mu         sync.Mutex
	provider   Provider
	cookieName string
	maxAge     int64
	adapter    AgeCheckerAdapter
}

// Returns a new Manager (address for pointer reference).
//
// The provider cannot be nil and cookie name cannot be empty.
func newManager(provider Provider, cookieName string, maxAge int64, adapter AgeCheckerAdapter) *Manager {
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
	// go func(ctx context.Context) {
	// 	<-ctx.Done()
	// 	fmt.Println("SAVE SESSION AT THIS TIME")
	// }(r.Context())
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
	m.provider.SessionGC(m.adapter(m.maxAge))
	time.AfterFunc(time.Duration(m.maxAge), func() {
		m.GC()
	})
}

var manager *Manager

func Reset(cookieName string, maxAge int64, adapter AgeCheckerAdapter, storage Storage) {
	manager = newManager(
		newProvider(storage),
		cookieName,
		maxAge,
		adapter,
	)
}

func Start(w http.ResponseWriter, r *http.Request) Session {
	return manager.StartSession(w, r)
}

func Destroy(w http.ResponseWriter, r *http.Request) {
	manager.DestroySession(w, r)
}
