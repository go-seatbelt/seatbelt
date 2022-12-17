package render

import (
	"bytes"
	"html/template"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestRender(t *testing.T) {
	r := New(&Options{
		Dir:    filepath.Join("testdata", "templates"),
		Layout: "layout",
		Funcs: []ContextualFuncMap{
			func(w http.ResponseWriter, r *http.Request) template.FuncMap {
				return map[string]interface{}{
					"path": func() string {
						return r.URL.Path
					},
				}
			},
		},
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
			r.HTML(w, req, "home", nil)
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

	t.Run("render a non-existent template should 200 with an error message", func(t *testing.T) {
		rr := httptest.NewRecorder()
		tr := httptest.NewRequest(http.MethodGet, "/", nil)

		h := func(w http.ResponseWriter, req *http.Request) {
			r.HTML(w, req, "not-found.html", nil)
		}

		h(rr, tr)

		code := rr.Result().StatusCode
		expected := 200
		if code != expected {
			t.Errorf("expected status code %d but got %d", expected, code)
		}

		s := rr.Body.String()
		contains := `html/template: "not-found.html" is undefined`
		if !strings.Contains(s, contains) {
			t.Errorf("expected body %s to contain %s", s, contains)
		}
	})

	t.Run("render a plaintext error", func(t *testing.T) {
		rr := httptest.NewRecorder()
		tr := httptest.NewRequest(http.MethodGet, "/", nil)

		h := func(w http.ResponseWriter, req *http.Request) {
			r.TextError(rr, "texterror", 500)
		}

		h(rr, tr)

		code := rr.Result().StatusCode
		expected := 500
		if code != expected {
			t.Errorf("expected status code %d but got %d", expected, code)
		}

		s := rr.Body.String()
		contains := "texterror"
		if !strings.Contains(s, contains) {
			t.Errorf("expected body %s to contain %s", s, contains)
		}
	})

	t.Run("render a template with contextual funcs", func(t *testing.T) {
		rr := httptest.NewRecorder()
		tr := httptest.NewRequest(http.MethodGet, "/home", nil)

		h := func(w http.ResponseWriter, req *http.Request) {
			r.HTML(rr, tr, "home", nil)
		}

		h(rr, tr)

		code := rr.Result().StatusCode
		expected := 200
		if code != expected {
			t.Errorf("expected status code %d but got %d", expected, code)
		}

		s := rr.Body.String()
		contains := ""
		if !strings.Contains(s, contains) {
			t.Errorf("expected body %s to contain %s", s, contains)
		}
	})
}

func BenchmarkRender(b *testing.B) {
	r := New(&Options{
		Dir:    filepath.Join("testdata", "templates"),
		Reload: false,
		Funcs: []ContextualFuncMap{
			func(w http.ResponseWriter, r *http.Request) template.FuncMap {
				return map[string]interface{}{
					"path": func() string {
						return r.URL.Path
					},
				}
			},
		},
	})
	w := io.Discard

	for i := 0; i < b.N; i++ {
		r.HTML(w, nil, "index", nil)
	}
}

func BenchmarkSlowRender(b *testing.B) {
	r := New(&Options{
		Dir:    filepath.Join("testdata", "templates"),
		Layout: "layout",
		Reload: true,
		Funcs: []ContextualFuncMap{
			func(w http.ResponseWriter, r *http.Request) template.FuncMap {
				return map[string]interface{}{
					"path": func() string {
						return r.URL.Path
					},
				}
			},
		},
	})
	w := io.Discard

	for i := 0; i < b.N; i++ {
		r.HTML(w, nil, "index", nil)
	}
}
