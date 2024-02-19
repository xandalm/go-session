package session_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/xandalm/go-session"
)

type mockServer struct {
	manager *session.Manager
	players []string
}

func newServer(manager *session.Manager) *mockServer {
	return &mockServer{
		manager,
		make([]string, 0),
	}
}

func (s *mockServer) ServerHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/login":
		s.handleLogIn(w, r)
	case "/players":
		s.handleGetOnlinePlayers(w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (s *mockServer) handleLogIn(w http.ResponseWriter, r *http.Request) {
	session := s.manager.StartSession(w, r)
	if session == nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	username := r.FormValue("username")
	if username == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	err := session.Set("username", username)
	if err != nil {
		s.manager.DestroySession(w, r)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	err = session.Set("logged", true)
	if err != nil {
		s.manager.DestroySession(w, r)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	s.players = append(s.players, username)
}

func (s *mockServer) handleGetOnlinePlayers(w http.ResponseWriter, r *http.Request) {
	session := s.manager.StartSession(w, r)

	if isLogged, ok := session.Get("logged").(bool); !ok || !isLogged {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	fmt.Fprint(w, strings.Join(s.players, ","))
}

func TestSessionsWithMemoryStorage(t *testing.T) {
	storage := session.NewMemoryStorage()
	provider := session.NewDefaultProvider(session.DefaultSessionBuilder, storage, nil)
	manager := session.NewManager(provider, "SESSION_ID", 60)

	server := newServer(manager)

	var cookie *http.Cookie = nil

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

		request, _ := http.NewRequest(http.MethodPost, "http://foo.com/login", strings.NewReader(form.Encode()))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		response := httptest.NewRecorder()

		server.ServerHTTP(response, request)

		assertHTTPStatus(t, response, http.StatusOK)
		cookie = parseCookie(getCookieFromResponse(response))
	})

	form := url.Values{}
	form.Set("username", "andre")
	request, _ := http.NewRequest(http.MethodPost, "http://foo.com/login", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	server.ServerHTTP(httptest.NewRecorder(), request)

	t.Run("get players", func(t *testing.T) {

		request, _ := http.NewRequest(http.MethodGet, "http://foo.com/players", nil)
		request.AddCookie(cookie)
		response := httptest.NewRecorder()

		server.ServerHTTP(response, request)

		assertHTTPStatus(t, response, http.StatusOK)
		got := response.Body.String()
		want := strings.Join(server.players, ",")

		if got != want {
			t.Errorf("got players %s, but want %s", got, want)
		}
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
