package session_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/xandalm/go-session"
	"github.com/xandalm/go-session/memory"
)

type mockServer struct {
	players []string
}

func newServer() *mockServer {
	return &mockServer{
		make([]string, 0),
	}
}

func (s *mockServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	sess := session.Start(w, r)
	if sess == nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	switch r.URL.Path {
	case "/login":
		s.handleLogIn(w, r, sess)
	case "/players":
		s.handleGetOnlinePlayers(w, r, sess)
	case "/start":
		s.handleStartGame(w, r, sess)
	case "/leave":
		s.handleLeaveGame(w, r, sess)
	case "/score":
		s.handleScore(w, r, sess)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (s *mockServer) handleLogIn(w http.ResponseWriter, r *http.Request, sess session.Session) {
	username := r.FormValue("username")
	if username == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	err := sess.Set("username", username)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	err = sess.Set("logged", true)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	s.players = append(s.players, username)
}

func (s *mockServer) handleGetOnlinePlayers(w http.ResponseWriter, _ *http.Request, sess session.Session) {

	if isLogged, ok := sess.Get("logged").(bool); !ok || !isLogged {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	fmt.Fprint(w, strings.Join(s.players, ","))
}

func (s *mockServer) handleStartGame(w http.ResponseWriter, _ *http.Request, sess session.Session) {
	logged, ok := sess.Get("logged").(bool)
	if !ok || !logged {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	err := sess.Set("score", 0)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (s *mockServer) handleLeaveGame(w http.ResponseWriter, _ *http.Request, sess session.Session) {
	logged, ok := sess.Get("logged").(bool)
	if !ok || !logged {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	err := sess.Delete("score")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (s *mockServer) handleScore(w http.ResponseWriter, r *http.Request, sess session.Session) {
	logged, ok := sess.Get("logged").(bool)
	if !ok || !logged {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	score, ok := sess.Get("score").(int)
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if r.Method == http.MethodGet {
		fmt.Fprint(w, score)
		return
	}

	if err := sess.Set("score", score+1); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

type stubCookieManager struct {
	cookies map[string]*struct {
		v              *http.Cookie
		creationTime   time.Time
		lastAccessTime time.Time
	}
}

func newStubCookieManager() *stubCookieManager {
	return &stubCookieManager{
		make(map[string]*struct {
			v              *http.Cookie
			creationTime   time.Time
			lastAccessTime time.Time
		}),
	}
}

func (m *stubCookieManager) expires(cookie *struct {
	v              *http.Cookie
	creationTime   time.Time
	lastAccessTime time.Time
}) bool {
	maxAge := cookie.v.MaxAge
	expires := cookie.v.Expires
	if maxAge < 0 {
		return true
	}
	if !expires.IsZero() {
		return expires.Before(time.Now())
	}
	return time.Now().Unix()-cookie.creationTime.Unix() > int64(maxAge)
}

func (m *stubCookieManager) SetCookie(cookie *http.Cookie) {
	if c, ok := m.cookies[cookie.Name]; ok {
		c.v = cookie
		c.lastAccessTime = time.Now()
		return
	}
	m.cookies[cookie.Name] = &struct {
		v              *http.Cookie
		creationTime   time.Time
		lastAccessTime time.Time
	}{cookie, time.Now(), time.Now()}
}

func (m *stubCookieManager) Cookies() []*http.Cookie {
	keep := make(map[string]*struct {
		v              *http.Cookie
		creationTime   time.Time
		lastAccessTime time.Time
	})
	ret := []*http.Cookie{}
	for name, c := range m.cookies {
		if m.expires(c) {
			continue
		}
		keep[name] = c
		ret = append(ret, c.v)
	}
	m.cookies = keep
	return ret
}

func performTest(t *testing.T) {
	t.Helper()

	server := newServer()

	cookieManager := newStubCookieManager()

	parseCookie := func(cookie map[string]string) *http.Cookie {
		maxAge, _ := strconv.Atoi(cookie["Max-Age"])
		httpOnly, _ := strconv.ParseBool(cookie["HttpOnly"])
		c := &http.Cookie{
			Name:     "SESSION_ID",
			Value:    cookie["SESSION_ID"],
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

	t.Run("login", func(t *testing.T) {

		form := url.Values{}
		form.Set("username", "alex")

		response := doPost(server, "http://foo.com/login", strings.NewReader(form.Encode()), map[string]string{"Content-Type": "application/x-www-form-urlencoded"}, nil)

		assertHTTPStatus(t, response, http.StatusOK)
		cookie := parseCookie(getCookieFromResponse(response))
		cookieManager.SetCookie(cookie)
	})

	form := url.Values{}
	form.Set("username", "andre")

	doPost(server, "http://foo.com/login", strings.NewReader(form.Encode()), map[string]string{"Content-Type": "application/x-www-form-urlencoded"}, nil)

	t.Run("get players", func(t *testing.T) {

		response := doGet(server, "http://foo.com/players", nil, nil, cookieManager.Cookies())

		assertHTTPStatus(t, response, http.StatusOK)
		got := response.Body.String()
		want := strings.Join(server.players, ",")

		if got != want {
			t.Errorf("got players %s, but want %s", got, want)
		}
	})

	t.Run("start game", func(t *testing.T) {

		response := doPost(server, "http://foo.com/start", nil, nil, cookieManager.Cookies())

		assertHTTPStatus(t, response, http.StatusOK)
	})

	t.Run("score in the game", func(t *testing.T) {

		response := doPost(server, "http://foo.com/score", nil, nil, cookieManager.Cookies())

		assertHTTPStatus(t, response, http.StatusOK)
	})

	t.Run("get score", func(t *testing.T) {

		response := doGet(server, "http://foo.com/score", nil, nil, cookieManager.Cookies())

		assertHTTPStatus(t, response, http.StatusOK)

		got := response.Body.String()
		want := "1"

		if got != want {
			t.Errorf("got score %s, but want %s", got, want)
		}
	})

	t.Run("leave from game", func(t *testing.T) {

		response := doPost(server, "http://foo.com/leave", nil, nil, cookieManager.Cookies())

		assertHTTPStatus(t, response, http.StatusOK)
	})

	t.Run("logoff player after session expires", func(t *testing.T) {
		time.Sleep(2 * time.Second)

		response := doGet(server, "http://foo.com/players", nil, nil, cookieManager.Cookies())

		assertHTTPStatus(t, response, http.StatusUnauthorized)
	})
}

func assertHTTPStatus(t testing.TB, response *httptest.ResponseRecorder, want int) {
	t.Helper()

	got := response.Code
	if got != want {
		t.Fatalf("got http status %d, but want %d", got, want)
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

func doGet(handler http.Handler, url string, body io.Reader, headers map[string]string, cookies []*http.Cookie) *httptest.ResponseRecorder {
	ctx, cancel := context.WithCancel(context.Background())

	request, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, body)

	for k, v := range headers {
		request.Header.Set(k, v)
	}

	for _, c := range cookies {
		request.AddCookie(c)
	}

	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)
	cancel()

	return response
}

func doPost(handler http.Handler, url string, body io.Reader, headers map[string]string, cookies []*http.Cookie) *httptest.ResponseRecorder {
	ctx, cancel := context.WithCancel(context.Background())

	request, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, body)

	for k, v := range headers {
		request.Header.Set(k, v)
	}

	for _, c := range cookies {
		request.AddCookie(c)
	}

	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)
	cancel()

	return response
}

func TestSessionsWithMemoryStorage(t *testing.T) {
	session.Reset("SESSION_ID", 1, session.SecondsAgeCheckerAdapter, memory.NewStorage())

	performTest(t)
}

// func TestSessionsWithFileSystemStorage(t *testing.T) {
// 	path := "sessions_from_integration_test"
// 	provider := session.NewProvider(filesystem.Storage(path), session.SecondsAgeCheckerAdapter)
// 	manager := session.NewManager(provider, "SESSION_ID", 1)

// 	performTest(t, manager)

// 	t.Cleanup(func() {
// 		if err := os.RemoveAll(path); err != nil {
// 			log.Fatalf("cannot clean up after test, %v", err)
// 		}
// 	})
// }
