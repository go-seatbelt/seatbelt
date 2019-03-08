package controllers_test

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/go-seatbelt/seatbelt"
	"github.com/go-seatbelt/seatbelt/controllers"
	"github.com/go-seatbelt/seatbelt/internal/trace"
)

func TestController(t *testing.T) {
	t.Parallel()

	// Set the basepath, since we're executing HTML in this test.
	basepath := filepath.Dir(filepath.Join(trace.File(), ".", ".."))

	render := &controllers.Render{}
	render.Basepath = basepath

	c := controllers.Controller{Render: render}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/test", nil)

	value := "value"

	c.HTML(w, r, "test/index", seatbelt.Data{
		"Key": value,
	})

	body := w.Body.String()
	expected := `<h1>` + value + `</h1>`

	if !strings.Contains(body, expected) {
		t.Fatalf("expected body to contain %s, but got %s", expected, body)
	}
}

func TestControllerFlashes(t *testing.T) {
	t.Parallel()

	c := controllers.New()
	w := httptest.NewRecorder()
	r, err := http.NewRequest(http.MethodGet, "/test", nil)
	if err != nil {
		t.Fatalf("error creating request: %+v", err)
	}

	flash := controllers.Flash{"key": "value"}
	c.Flash(w, r, flash)

	for _, cookie := range w.Result().Cookies() {
		r.AddCookie(cookie)
	}

	foundFlash := c.GetFlash(w, r)
	if !reflect.DeepEqual(flash, foundFlash) {
		t.Fatalf("expected flash messages to be equal, but got %+v vs %+v", flash, foundFlash)
	}
}
