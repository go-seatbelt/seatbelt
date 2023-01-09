package values

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestValues(t *testing.T) {
	v := New(httptest.NewRequest(http.MethodGet, "/", nil))
	expected := "value"

	t.Run("set and get", func(t *testing.T) {
		v.Set("key", expected)

		actual := v.Get("key").(string)

		if expected != actual {
			t.Fatalf("expected %s but got %s", expected, actual)
		}
	})

	t.Run("list", func(t *testing.T) {
		vs := v.List()

		if n := len(vs); n != 1 {
			t.Fatalf("expected length 1 but got %d", n)
		}

		actual := vs["key"].(string)

		if expected != actual {
			t.Fatalf("expected %s but got %s", expected, actual)
		}
	})

	t.Run("delete", func(t *testing.T) {
		v.Delete("key")

		data := v.List()

		if n := len(data); n != 0 {
			t.Fatalf("expected length to be zero but got %d", n)
		}
	})
}
