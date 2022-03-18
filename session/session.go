package session

import (
	"context"
	"log"
	"net/http"

	"github.com/gorilla/securecookie"
)

type sessionCtxKeyType struct{}

const defaultSessionName = "_session"

var sessionCtxKey = sessionCtxKeyType{}

// A Session manages setting and getting data from the cookie that stores the
// session data.
type Session struct {
	sc *securecookie.SecureCookie
}

// New creates a new session with the given key.
func New(secret []byte) *Session {
	return &Session{
		sc: securecookie.New(secret, nil),
	}
}

// fromReq returns the map of session values from the request. It will
// never return a nil map, instead, the map will be an initialized empty map
// in the case where the session has no data.
func (s *Session) fromReq(r *http.Request) map[string]interface{} {
	// Fastpath: if the context has already been decoded, access the
	// underlying map and return the value associated with the given key.
	v := r.Context().Value(sessionCtxKey)
	if v != nil {
		data, ok := v.(map[string]interface{})
		if ok {
			return data
		}
	}

	cookie, err := r.Cookie(defaultSessionName)
	if err != nil {
		// The only error that can be returned by r.Cookie() is ErrNoCookie,
		// so if the error is not nil, that means that the cookie doesn't
		// exist. When that is the case, the value associated with the given
		// key is guaranteed to be nil, so we return nil.
		return make(map[string]interface{})
	}

	data := make(map[string]interface{})
	if err := s.sc.Decode(defaultSessionName, cookie.Value, &data); err != nil {
		log.Println("[error] failed to decode session from cookie:", err)
		return make(map[string]interface{})
	}
	return data
}

// saveCtx saves a map of session data in the current request's context. It
// also updates the Set-Cookie header of the
func (s *Session) saveCtx(w http.ResponseWriter, r *http.Request, data map[string]interface{}) {
	ctx := context.WithValue(r.Context(), sessionCtxKey, data)
	r2 := r.Clone(ctx)
	*r = *r2

	encoded, err := s.sc.Encode(defaultSessionName, data)
	if err != nil {
		log.Println("error encoding cookie:", err)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     defaultSessionName,
		Value:    encoded,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
	})
}

// Session creates a new session from the given HTTP request. If the
// request already has a cookie with an associated session, the session data
// is created from the cookie. If not, a new session is created.
func (s *Session) Get(r *http.Request, key string) interface{} {
	data := s.fromReq(r)
	return data[key]
}

// List returns all key value pairs of session data from the given request.
func (s *Session) List(r *http.Request) map[string]interface{} {
	return s.fromReq(r)
}

// Set sets or updates the given value on the session.
func (s *Session) Set(w http.ResponseWriter, r *http.Request, key string, value interface{}) {
	data := s.fromReq(r)
	data[key] = value
	s.saveCtx(w, r, data)
}

// Delete deletes and returns the session value with the given key.
func (s *Session) Delete(w http.ResponseWriter, r *http.Request, key string) interface{} {
	data := s.fromReq(r)
	value := data[key]
	delete(data, key)
	s.saveCtx(w, r, data)
	return value
}
