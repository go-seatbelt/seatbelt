package i18n

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTranslator(t *testing.T) {
	translator := New("testdata", false)

	req := httptest.NewRequest(http.MethodGet, "/?locale=fr", nil)

	s := translator.T(req, "PersonCats", map[string]interface{}{
		"Name":  "Ben",
		"Count": 0,
	})

	expected := "Ben a 0 chats."
	if expected != s {
		t.Fatalf("expected %s but got %s", expected, s)
	}
}
