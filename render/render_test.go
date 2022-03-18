package render

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestRender(t *testing.T) {
	r := New(Options{
		Dir: filepath.Join("testdata", "templates"),
	})

	t.Run("render a template successfully to an io.Writer", func(t *testing.T) {
		b := &bytes.Buffer{}

		r.HTML(b, nil, "index", nil)

		s := b.String()
		contains := "<!DOCTYPE html>"
		if !strings.Contains(s, contains) {
			t.Errorf("expected %s to contain %s", s, contains)
		}
	})

	t.Run("render a template successfully to an http.ResponseWriter", func(t *testing.T) {
		rr := httptest.NewRecorder()
		tr := httptest.NewRequest(http.MethodGet, "/", nil)

		h := func(w http.ResponseWriter, req *http.Request) {
			r.HTML(w, req, "index", nil)
		}

		h(rr, tr)

		code := rr.Result().StatusCode
		expected := http.StatusOK
		if code != expected {
			t.Errorf("expected status code %d but got %d", expected, code)
		}

		s := rr.Body.String()
		contains := "<!DOCTYPE html>"
		if !strings.Contains(s, contains) {
			t.Errorf("expected body %s to contain %s", s, contains)
		}
	})

	t.Run("render a non-existent template should 500", func(t *testing.T) {
		rr := httptest.NewRecorder()
		tr := httptest.NewRequest(http.MethodGet, "/", nil)

		h := func(w http.ResponseWriter, req *http.Request) {
			r.HTML(w, req, "not-found.html", nil)
		}

		h(rr, tr)

		code := rr.Result().StatusCode
		expected := http.StatusInternalServerError
		if code != expected {
			t.Errorf("expected status code %d but got %d", expected, code)
		}

		s := rr.Body.String()
		contains := "no template"
		if !strings.Contains(s, contains) {
			t.Errorf("expected body %s to contain %s", s, contains)
		}
	})
}

func BenchmarkRender(b *testing.B) {
	r := New(Options{
		Dir: filepath.Join("testdata", "templates"),
	})
	w := io.Discard

	for i := 0; i < b.N; i++ {
		r.HTML(w, nil, "index", nil)
	}
}

func BenchmarkSlowRender(b *testing.B) {
	r := New(Options{
		Reload: true,
		Dir:    filepath.Join("testdata", "templates"),
	})
	w := io.Discard

	for i := 0; i < b.N; i++ {
		r.HTML(w, nil, "index", nil)
	}
}
