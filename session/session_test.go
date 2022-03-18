package session

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/csrf"
	"github.com/gorilla/securecookie"
)

func TestSessionManager(t *testing.T) {
	s := New(securecookie.GenerateRandomKey(32))

	t.Run("get should never return a nil session", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		v := s.Get(req, "notfound")

		if v != nil {
			t.Fatalf("expected session value to be nil but got %#v", v)
		}
	})

	t.Run("save session data to the current request", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)

		s.Set(rr, req, "key", "value")
		v := s.Get(req, "key")

		if v == nil {
			t.Fatal("expected session value not to be nil")
		}
		if s := v.(string); s != "value" {
			t.Fatalf("expected value to be value but got %s", s)
		}

		if rr.Result().Header.Get("Set-Cookie") == "" {
			t.Fatal("expected Set-Cookie header but got empty string")
		}
	})
}

func TestSessionManagerIntegration(t *testing.T) {
	secret := securecookie.GenerateRandomKey(32)
	s := New(secret)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Set(w, r, "key", "value")
		w.Write([]byte("Hello, world!"))
	})

	srv1 := httptest.NewServer(handler)
	srv2 := httptest.NewServer(csrf.Protect(secret)(handler))
	defer srv1.Close()
	defer srv2.Close()

	cases := []struct {
		name string
		srv  *httptest.Server
	}{
		{
			name: "setting a cookie should apply the set cookie header",
			srv:  srv1,
		},
		{
			name: "setting a cookie should apply the set cookie header even with middleware that sets a cookie",
			srv:  srv2,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			resp, err := http.Get(c.srv.URL + "/")
			if err != nil {
				t.Fatalf("error getting /: %+v", err)
			}

			expected := http.StatusOK
			if resp.StatusCode != expected {
				t.Fatalf("expected status %d but got %d", expected, resp.StatusCode)
			}

			rawcookies := resp.Header["Set-Cookie"]
			if rawcookies == nil {
				t.Fatalf("missing set cookie header")
			}

			found := false
			for _, rawcookie := range rawcookies {
				if strings.HasPrefix(rawcookie, "_session=") {
					found = true
				}
			}
			if !found {
				t.Fatalf("expected to find cookie with name _session in %s", strings.Join(rawcookies, ","))
			}
		})
	}
}
