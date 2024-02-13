package session

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/xandalm/go-session/testing/assert"
)

var dummySite = "http://site.com"

func TestManager(t *testing.T) {
	cookieName := "SessionID"
	provider := &stubProvider{}
	manager := NewManager(provider, cookieName, 3600)

	assert.AssertNotNil(t, manager)

	var cookie *http.Cookie = nil

	parseCookie := func(cookie map[string]string) *http.Cookie {
		maxAge, _ := strconv.Atoi(cookie["Max-Age"])
		httpOnly, _ := strconv.ParseBool(cookie["HttpOnly"])
		c := &http.Cookie{
			Name:     cookieName,
			Value:    cookie[cookieName],
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

	t.Run("start the session", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, dummySite, nil)
		res := httptest.NewRecorder()

		session := manager.StartSession(res, req)
		assert.AssertNotNil(t, session)

		cookie = parseCookie(getCookieFromResponse(res))
		assert.AssertNotNil(t, cookie)
		cookie.Name = cookieName

		sid := cookie.Value

		assert.AssertNotEmpty(t, sid)

		assert.AssertEqual(t, sid, url.QueryEscape(session.SessionID()))
	})

	t.Run("restores the same session", func(t *testing.T) {

		req, _ := http.NewRequest(http.MethodGet, dummySite+"/admin", nil)
		req.AddCookie(cookie)

		res := httptest.NewRecorder()

		session := manager.StartSession(res, req)
		assert.AssertNotNil(t, session)

		assert.AssertEqual(t, cookie.Value, url.QueryEscape(session.SessionID()))
	})

	t.Run("destroy the session", func(t *testing.T) {

		req, _ := http.NewRequest(http.MethodGet, dummySite, nil)
		req.AddCookie(cookie)

		res := httptest.NewRecorder()

		manager.DestroySession(res, req)

		sid, _ := url.QueryUnescape(cookie.Value)

		if _, ok := provider.Sessions[sid]; ok {
			t.Fatalf("didn't destroy session")
		}

		newCookie := parseCookie(getCookieFromResponse(res))
		assert.AssertNotNil(t, newCookie)

		if newCookie.Expires.After(time.Now()) || newCookie.MaxAge != 0 {
			t.Errorf("the cookie is not expired, Expires = %s and MaxAge = %d", newCookie.Expires, newCookie.MaxAge)
		}
	})
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
