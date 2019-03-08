package server_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-seatbelt/seatbelt/server"
)

func TestTurbo(t *testing.T) {
	t.Run("turbolinks redirect", func(t *testing.T) {
		h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/redirect", http.StatusFound)
		})

		res := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/", nil)

		// Set the header to make sure we hit the Turbolinks handler.
		req.Header.Set("Turbolinks-Referrer", "http://localhost:3000/redirect")
		turboh := server.Turbolinks(h)
		turboh.ServeHTTP(res, req)

		if res.Code != http.StatusFound {
			t.Fatalf("expected HTTP status %d but got %d", http.StatusFound, res.Code)
		}

		cookieReq := &http.Request{Header: http.Header{"Cookie": res.HeaderMap["Set-Cookie"]}}
		_, err := cookieReq.Cookie(server.TurbolinksCookie)
		if err != nil {
			t.Fatalf("expected cookie but got %v", err.Error())
		}
	})

	t.Run("turbolinks form submission", func(t *testing.T) {
		h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/", http.StatusFound)
		})

		res := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/", nil)

		// Set the header to make sure we hit the Turbolinks handler.
		req.Header.Set("Turbolinks-Referrer", "http://localhost:3000/redirect")
		turboh := server.Turbolinks(h)
		turboh.ServeHTTP(res, req)

		if res.Code != http.StatusOK {
			t.Fatalf("expected HTTP status %d but got %d", http.StatusOK, res.Code)
		}
		contentType := res.Header().Get("Content-Type")
		if contentType != "text/javascript" {
			t.Fatalf("expected Content-Type to be text/javascript but got %s", contentType)
		}
		expectedJS := `Turbolinks.clearCache();Turbolinks.visit("/", {action: "advance"});`
		actualJS := res.Body.String()
		if actualJS != expectedJS {
			t.Fatalf("expected response to be %s but got %s", expectedJS, actualJS)
		}
	})
}
