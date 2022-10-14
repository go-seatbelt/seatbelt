// Package handler provides convenience methods for working with HTTP
// requests.
package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/mitchellh/mapstructure"
)

const (
	// defaultMaxMemory is the default max memory for parsing forms, including
	// multipart forms. This is the same as the default value used in the
	// standard library.
	defaultMaxMemory = 32 << 20 // 32 MB
)

// Redirect replies to the request with a redirect to url, which may be a path
// relative to the request path.
//
// The code is "found" for any idempotent request, and "see other" for all
// others.
//
// If the Content-Type header has not been set, Redirect sets it to
// "text/html; charset=utf-8" and writes a small HTML body. Setting the
// Content-Type header to any value, including nil, disables that behaviour.
func Redirect(w http.ResponseWriter, r *http.Request, url string) {
	code := http.StatusFound

	if r.Method == http.MethodPost ||
		r.Method == http.MethodPut ||
		r.Method == http.MethodPatch ||
		r.Method == http.MethodDelete {
		code = http.StatusSeeOther
	}

	http.Redirect(w, r, url, code)
}

// JSON sends a JSON response with the given status code.
func JSON(w http.ResponseWriter, code int, v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.WriteHeader(code)

	_, err = w.Write(data)
	return err
}

// A PathParamFunc should parse the path params from the given request r, and
// assign them to the map v.
//
// For example, if you're using the Chi router to parse the path parameters,
// this method will look like:
//
//	if rctx := chi.RouteContext(r.Context()); rctx != nil {
//		for i, key := range rctx.URLParams.Keys {
//			values[key] = rctx.URLParams.Values[i]
//		}
//	}
//
// The top-level "seatbelt" package contains some PathParamFunc's for
// different routers, i.e., users of github.com/go-chi/chi can use
//
//	seatbelt.ChiPathParamFunc
//
// as an argument in the Params function.
type PathParamFunc func(r *http.Request, v map[string]interface{})

// Params mass-assigns query, path, and form parameters to the given struct or
// map, similar to how Rails mass-assignment works.
//
// v must be a pointer to a struct or a map.
//
// The precedence is as follows:
//
// 1. Path params (highest).
//
// 2. Body params.
//
// 3. Query params.
//
// For POST, PUT, PATCH, and DELETE requests, the body will be read. For any
// other request, it will not.
//
// See also the GoDoc string for PathParamFunc.
func Params(w http.ResponseWriter, r *http.Request, pathParamFunc PathParamFunc, v interface{}) error {
	var err error
	if r.Header.Get("Content-Type") == "multipart/form-data" {
		err = r.ParseMultipartForm(defaultMaxMemory)
	} else {
		err = r.ParseForm()
	}
	if err != nil {
		return err
	}

	// mapstructure doesn't like the map[string][]string that the query and
	// form data is in, so we turn it into a map[string]string.
	values := make(map[string]interface{})

	// Parse the body query parameters using the built-in Form map, as calling
	// ParseForm() already does what we want to do.
	for key, val := range r.Form {
		values[key] = strings.Join(val, "")
	}

	// Parse the JSON body if the content type and HTTP verb correct.
	if r.Method == http.MethodPost ||
		r.Method == http.MethodPut ||
		r.Method == http.MethodPatch ||
		r.Method == http.MethodDelete {
		if r.Header.Get("Content-Type") == "application/json" {
			if err := json.NewDecoder(r.Body).Decode(&values); err != nil {
				return err
			}
			defer r.Body.Close()
		}
	}

	// Finally, overwrite any values with path params.
	if pathParamFunc != nil {
		pathParamFunc(r, values)
	}

	// The config below is the same as mapstructure's `WeakDecode`, but with
	// the tag name "params" instead of "mapstructure".
	config := &mapstructure.DecoderConfig{
		Metadata:         nil,
		Result:           v,
		WeaklyTypedInput: true,
		TagName:          "params",
	}

	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		return err
	}

	return decoder.Decode(values)
}
