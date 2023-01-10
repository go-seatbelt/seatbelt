package seatbelt

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestOptions(t *testing.T) {
	o := &Option{}

	t.Run("a master.key file should be present after calling setDefaults", func(t *testing.T) {
		o.setDefaults()

		data, err := os.ReadFile("master.key")
		if err != nil {
			t.Fatalf("failed to read master.key file: %v", err)
		}
		if data == nil {
			t.Fatal("file is empty")
		}
	})
}

func TestSubRouter(t *testing.T) {
	app := New()

	app.Get("/", func(c *Context) error {
		return c.String(200, "home")
	})
	app.Namespace("/admin", func(app *App) {
		app.Get("/home", func(c *Context) error {
			return c.String(200, "ok")
		})
	})

	srv := httptest.NewServer(app)
	defer srv.Close()

	t.Run("GET /", func(t *testing.T) {
		resp, err := http.Get(srv.URL + "/")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		data, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}

		if string(data) != "home" {
			t.Fatalf("expected home but got %s", data)
		}
	})

	t.Run("GET /admin/home", func(t *testing.T) {
		resp, err := http.Get(srv.URL + "/admin/home")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		data, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}

		if string(data) != "ok" {
			t.Fatalf("expected ok but got %s", data)
		}
	})
}
