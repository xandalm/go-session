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
	"github.com/xandalm/go-session/filesystem"
)

type stubServer struct {
	players []string
}

func newServer() *stubServer {
	return &stubServer{
		make([]string, 0),
	}
}

func (s *stubServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	sess := session.Start(w, r)
	if sess == nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	switch r.URL.Path {
	case "/login":
		s.handleLogIn(w, r, sess)
	case "/logout":
		session.Destroy(w, r)
		w.WriteHeader(http.StatusOK)
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

func (s *stubServer) handleLogIn(w http.ResponseWriter, r *http.Request, sess session.Session) {
	username := r.FormValue("username")
	if username == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if err := sess.Set("username", username); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if err := sess.Set("logged", true); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	s.players = append(s.players, username)
}

func (s *stubServer) handleGetOnlinePlayers(w http.ResponseWriter, _ *http.Request, sess session.Session) {
	if isLogged, ok := sess.Get("logged").(bool); !ok || !isLogged {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	fmt.Fprint(w, strings.Join(s.players, ","))
}

func (s *stubServer) handleStartGame(w http.ResponseWriter, _ *http.Request, sess session.Session) {
	logged, ok := sess.Get("logged").(bool)
	if !ok || !logged {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	sess.Set("score", 0)
}

func (s *stubServer) handleLeaveGame(w http.ResponseWriter, _ *http.Request, sess session.Session) {
	logged, ok := sess.Get("logged").(bool)
	if !ok || !logged {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	sess.Delete("score")
}

func (s *stubServer) handleScore(w http.ResponseWriter, r *http.Request, sess session.Session) {
	logged, ok := sess.Get("logged").(bool)
	if !ok || !logged {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	score, ok := sess.Get("score").(int)
	if !ok {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	if r.Method == http.MethodGet {
		fmt.Fprint(w, score)
		return
	}

	sess.Set("score", score+1)
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

const sessionCookieName = "SESSION_ID"

func parseCookie(cookie map[string]string) *http.Cookie {
	maxAge, _ := strconv.Atoi(cookie["Max-Age"])
	httpOnly, _ := strconv.ParseBool(cookie["HttpOnly"])
	c := &http.Cookie{
		Name:     sessionCookieName,
		Value:    cookie[sessionCookieName],
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

func performTest(t *testing.T) {
	t.Helper()

	server := newServer()

	t.Run("login", func(t *testing.T) {

		cookieManager := newStubCookieManager()

		form := url.Values{}
		form.Set("username", "alex")

		body := strings.NewReader(form.Encode())
		response := doPost(server, "http://foo.com/login", body, map[string]string{"Content-Type": "application/x-www-form-urlencoded"}, nil)

		assertHTTPStatus(t, response, http.StatusOK)
		cookie := parseCookie(getCookieFromResponse(response))
		cookieManager.SetCookie(cookie)

		t.Run("get players", func(t *testing.T) {

			cookies := cookieManager.Cookies()
			response := doGet(server, "http://foo.com/players", nil, nil, cookies)

			assertHTTPStatus(t, response, http.StatusOK)
			got := response.Body.String()
			want := "alex"

			if got != want {
				t.Errorf("got players %s, but want %s", got, want)
			}
		})

		t.Run("start game", func(t *testing.T) {

			cookies := cookieManager.Cookies()
			response := doPost(server, "http://foo.com/start", nil, nil, cookies)

			assertHTTPStatus(t, response, http.StatusOK)
		})

		t.Run("score in the game", func(t *testing.T) {

			cookies := cookieManager.Cookies()
			response := doPost(server, "http://foo.com/score", nil, nil, cookies)

			assertHTTPStatus(t, response, http.StatusOK)
		})

		t.Run("get score", func(t *testing.T) {

			cookies := cookieManager.Cookies()
			response := doGet(server, "http://foo.com/score", nil, nil, cookies)

			assertHTTPStatus(t, response, http.StatusOK)

			got := response.Body.String()
			want := "1"

			if got != want {
				t.Errorf("got score %s, but want %s", got, want)
			}
		})

		t.Run("leave from game", func(t *testing.T) {

			cookies := cookieManager.Cookies()
			response := doPost(server, "http://foo.com/leave", nil, nil, cookies)

			assertHTTPStatus(t, response, http.StatusOK)
		})

		t.Run("logout", func(t *testing.T) {

			cookies := cookieManager.Cookies()
			response := doGet(server, "http://foo.com/logout", nil, nil, cookies)

			assertHTTPStatus(t, response, http.StatusOK)

			response = doPost(server, "http://foo.com/start", nil, nil, cookies)
			assertHTTPStatus(t, response, http.StatusUnauthorized)
		})
	})

	cookieManager := newStubCookieManager()

	form := url.Values{}
	form.Set("username", "andre")

	response := doPost(server, "http://foo.com/login", strings.NewReader(form.Encode()), map[string]string{"Content-Type": "application/x-www-form-urlencoded"}, nil)

	assertHTTPStatus(t, response, http.StatusOK)
	cookie := parseCookie(getCookieFromResponse(response))
	cookieManager.SetCookie(cookie)

	t.Run("logoff player by expired session", func(t *testing.T) {

		time.Sleep(1 * time.Second)

		cookies := cookieManager.Cookies()
		response := doGet(server, "http://foo.com/start", nil, nil, cookies)

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

func TestSessionsWithFileSystemStorage(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	path := "sessions_from_integration_test"
	session.Config("SESSION_ID", 1, session.DefaultSessionFactory, filesystem.NewStorage(path, ""))

	performTest(t)
}

// func BenchmarkOnFileSystemStorage(b *testing.B) {
// 	path := "sessions_from_integration_test"
// 	session.ProviderSyncRoutineTime = 1 * time.Second
// 	session.Config("SESSION_ID", 3600, session.DefaultSessionFactory, filesystem.NewStorage(path, ""))
// 	h := newServer()

// 	init := func(username string) *stubCookieManager {
// 		cm := newStubCookieManager()
// 		form := url.Values{}
// 		form.Set("username", username)

// 		body := strings.NewReader(form.Encode())
// 		res := doPost(h, "http://foo.com/login", body, map[string]string{"Content-Type": "application/x-www-form-urlencoded"}, nil)
// 		assertHTTPStatus(b, res, http.StatusOK)

// 		cookie := parseCookie(getCookieFromResponse(res))
// 		cm.SetCookie(cookie)

// 		cookies := cm.Cookies()
// 		res = doPost(h, "http://foo.com/start", nil, nil, cookies)

// 		if res.Code != http.StatusOK {
// 			fmt.Println(username)
// 		}

// 		assertHTTPStatus(b, res, http.StatusOK)
// 		return cm
// 	}

// 	logout := func(cm *stubCookieManager) {
// 		cookies := cm.Cookies()
// 		res := doGet(h, "http://foo.com/logout", nil, nil, cookies)
// 		assertHTTPStatus(b, res, http.StatusOK)
// 	}

// 	score := func(cm *stubCookieManager, ch chan int8) {
// 		cookies := cm.Cookies()
// 		res := doPost(h, "http://foo.com/score", nil, nil, cookies)
// 		assertHTTPStatus(b, res, http.StatusOK)
// 		ch <- 1
// 	}

// 	players := []*stubCookieManager{
// 		init("john"),
// 		init("anie"),
// 		init("hugo"),
// 		init("riya"),
// 	}

// 	ch := make(chan int8, b.N*4)

// 	b.StartTimer()
// 	for i := 0; i < b.N; i++ {
// 		go score(players[0], ch)
// 		go score(players[1], ch)
// 		go score(players[2], ch)
// 		go score(players[3], ch)
// 	}
// 	for i := 0; i < b.N*4; i++ {
// 		<-ch
// 	}
// 	b.StopTimer()

// 	logout(players[0])
// 	logout(players[1])
// 	logout(players[2])
// 	logout(players[3])
// }
