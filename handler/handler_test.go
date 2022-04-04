package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-seatbelt/seatbelt/handler"
)

// expectEqual checks if the results are equal, and calls t.Fatal if not.
func expectEqual[T comparable](t *testing.T, expected, actual T) {
	if expected != actual {
		t.Fatalf("expected %v but got %v", expected, actual)
	}
}

func TestRedirect(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	handler.Redirect(w, r, "/")
	expectEqual(t, w.Result().StatusCode, http.StatusFound)
}

func TestParamsJSON(t *testing.T) {
	t.Parallel()

	s := &struct {
		Name string `params:"name"`
	}{}

	v := map[string]interface{}{
		"name": "test",
	}
	data, _ := json.Marshal(v)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(data))
	r.Header.Set("Content-Type", "application/json")

	if err := handler.Params(w, r, nil, s); err != nil {
		t.Fatal(err)
	}

	expectEqual(t, s.Name, "test")
}

func TestParamsURL(t *testing.T) {
	t.Parallel()

	s := &struct {
		Name string `params:"name"`
	}{}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/?name=test", nil)

	handler.Params(w, r, nil, s)

	expectEqual(t, s.Name, "test")
}

func TestParamsURLAndJSON(t *testing.T) {
	t.Parallel()

	s := &struct {
		Name       string `params:"name"`
		SecondName string `params:"second_name"`
	}{}

	v := map[string]interface{}{
		"name": "test",
	}
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/?second_name=test2", bytes.NewReader(data))
	r.Header.Set("Content-Type", "application/json")

	handler.Params(w, r, nil, s)

	expectEqual(t, s.Name, "test")
	expectEqual(t, s.SecondName, "test2")
}

func TestParamsURLAndJSONOverwrite(t *testing.T) {
	t.Parallel()

	s := &struct {
		Name string `params:"name"`
	}{}

	v := map[string]interface{}{
		"name": "test",
	}
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/?name=test2", bytes.NewReader(data))
	r.Header.Set("Content-Type", "application/json")

	handler.Params(w, r, nil, s)

	expectEqual(t, s.Name, "test")
}
