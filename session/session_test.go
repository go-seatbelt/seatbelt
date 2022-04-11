package session

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/csrf"
	"github.com/gorilla/securecookie"
)

func TestSessionGetNonNil(t *testing.T) {
	t.Parallel()

	s := New(securecookie.GenerateRandomKey(32))

	t.Run("get should never return a nil session", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		v := s.Get(req, "notfound")

		if v != nil {
			t.Fatalf("expected session value to be nil but got %#v", v)
		}
	})
}

func TestSessionSetGet(t *testing.T) {
	t.Parallel()

	s := New(securecookie.GenerateRandomKey(32))

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

func TestSessionList(t *testing.T) {
	t.Parallel()

	s := New(securecookie.GenerateRandomKey(32))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	s.Set(rr, req, "key1", "value1")
	s.Set(rr, req, "key2", "value2")

	sessionData := s.List(req)
	if v := sessionData["key1"].(string); v != "value1" {
		t.Fatalf("expected value1, got %s", v)
	}
	if v := sessionData["key2"].(string); v != "value2" {
		t.Fatalf("expected value2, got %s", v)
	}
}

func TestSessionDelete(t *testing.T) {
	t.Parallel()

	s := New(securecookie.GenerateRandomKey(32))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	s.Set(rr, req, "key", "value")
	v := s.Get(req, "key")
	if v == nil {
		t.Fatal("got nil")
	}
	str, ok := v.(string)
	if !ok {
		t.Fatal("failed to assert string type")
	}
	if str != "value" {
		t.Fatalf("expected value but got %s", str)
	}

	v = s.Delete(rr, req, "key")
	if v == nil {
		t.Fatal("got nil")
	}
	str, ok = v.(string)
	if !ok {
		t.Fatal("failed to assert string type")
	}
	if str != "value" {
		t.Fatalf("expected value but got %s", str)
	}

	v = s.Delete(rr, req, "key")
	if v != nil {
		t.Fatalf("expected deleted key to have nil value but got %v", v)
	}
}

func TestSessionFlashes(t *testing.T) {
	t.Parallel()

	s := New(securecookie.GenerateRandomKey(32))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	s.Flash(rr, req, "flash1", "value1")
	s.Flash(rr, req, "flash2", "value2")

	values1 := s.Flashes(rr, req)
	values2 := s.Flashes(rr, req)

	if v := values1["flash1"].(string); v != "value1" {
		t.Fatalf("expected value1, got %s", v)
	}
	if v := values1["flash2"].(string); v != "value2" {
		t.Fatalf("expected value2, got %s", v)
	}
	if len(values2) != 0 {
		t.Fatalf("expected empty initialized session but got %v", values2)
	}
}

func TestSessionManagerIntegration(t *testing.T) {
	t.Parallel()

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
