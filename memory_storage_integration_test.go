package session_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

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

func TestSessionsWithMemoryStorage(t *testing.T) {
	storage := session.NewMemoryStorage()
	provider := session.NewDefaultProvider(session.DefaultSessionBuilder, storage, nil)
	manager := session.NewManager(provider, "SESSION_ID", 60)

	server := newServer(manager)

	t.Run("login", func(t *testing.T) {

		form := url.Values{}
		form.Set("username", "alex")

		request, _ := http.NewRequest(http.MethodPost, "http://foo.com/login", strings.NewReader(form.Encode()))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		response := httptest.NewRecorder()

		server.ServerHTTP(response, request)

		assertHTTPStatus(t, response, http.StatusOK)
	})
}

func assertHTTPStatus(t testing.TB, response *httptest.ResponseRecorder, want int) {
	t.Helper()

	got := response.Code
	if got != want {
		t.Fatalf("got http status %d, but want %d", got, want)
	}
}
