package seatbelt

import (
	"net/http"

	"github.com/go-chi/chi"
	_ "golang.org/x/text/message" // Required for commands to work.
)

// KV is a shortcut for typing `map[string]interface{}`.
type KV map[string]interface{}

// ChiPathParamFunc extracts path paramters from the given HTTP request using
// the github.com/go-chi/chi router.
func ChiPathParamFunc(r *http.Request, values map[string]interface{}) {
	if rctx := chi.RouteContext(r.Context()); rctx != nil {
		for i, key := range rctx.URLParams.Keys {
			values[key] = rctx.URLParams.Values[i]
		}
	}
}
