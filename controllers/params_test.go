package controllers_test

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/go-seatbelt/seatbelt/controllers"
)

func TestParams(t *testing.T) {
	t.Parallel()

	t.Run("deocde a GET request into a model via query params", func(t *testing.T) {
		type v struct {
			Name string
			Age  int
		}

		expected := &v{
			Name: "Ben",
			Age:  25,
		}

		c := controllers.New()
		r := httptest.NewRequest(http.MethodGet, "/?name=Ben&age=25", nil)

		actual := &v{}
		if err := c.Params(r, actual); err != nil {
			t.Fatalf("error getting params: %+v", err)
		}
		if !reflect.DeepEqual(expected, actual) {
			t.Fatalf("expected %+v but got %+v", expected, actual)
		}
	})

	t.Run("decode a POST request into a model via query params", func(t *testing.T) {
		type v struct {
			Name string
			Age  int
		}

		expected := &v{
			Name: "Ben",
			Age:  25,
		}

		c := controllers.New()
		r := httptest.NewRequest(http.MethodPost, "/?name=Ben&age=25", nil)

		actual := &v{}
		if err := c.Params(r, actual); err != nil {
			t.Fatalf("error getting params: %+v", err)
		}
		if !reflect.DeepEqual(expected, actual) {
			t.Fatalf("expected %+v but got %+v", expected, actual)
		}
	})
}
