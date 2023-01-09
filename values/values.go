package values

import (
	"context"
	"net/http"
)

var valueskey = struct{}{}

type Values struct {
	r   *http.Request
	key interface{}
}

func New(r *http.Request) *Values {
	return &Values{
		r:   r,
		key: valueskey,
	}
}

// values returns the map saved on the current request's context, or creates
// it if it's not present.
func (v *Values) values() map[string]interface{} {
	m := v.r.Context().Value(v.key)
	if m == nil {
		// If m is nil, create the map, save it on the context, and return it.
		data := make(map[string]interface{})
		v.save(data)
		return data
	}

	data, ok := m.(map[string]interface{})
	if !ok {
		// TODO Log warning message, even though this code path shouldn't be
		// reachable.
		return map[string]interface{}{}
	}

	return data
}

// save saves the map on the current request context.
func (v *Values) save(m map[string]interface{}) {
	v.r = v.r.WithContext(context.WithValue(v.r.Context(), v.key, m))
}

// Set sets the given key value pair on the request. These values are
// passed to every HTML template by merging them with the given `data`.
func (v *Values) Set(key string, value any) {
	data := v.values()
	data[key] = value
	v.save(data)
}

// Get returns the request-scoped value with the given key.
func (v *Values) Get(key string) any {
	data := v.values()
	return data[key]
}

// List returns all request-scoped values.
func (v *Values) List() map[string]any {
	return v.values()
}

// Delete deletes the given request-scoped value.
func (v *Values) Delete(key string) {
	data := v.values()
	delete(data, key)
	v.save(data)
}
