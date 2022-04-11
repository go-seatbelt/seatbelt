package session

import (
	"context"
	"log"
	"net/http"

	"github.com/gorilla/securecookie"
)

type sessionCtxKeyType struct{}

const defaultSessionName = "_session"

var (
	sessionCtxKey = sessionCtxKeyType{}
)

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

// A session holds the session data. It contains two fields:
//
// - "data" for long-lived session data that persists between requests,
//
// - "flashes" for session data that should be deleted as soon as it is shown.
type session struct {
	Data    map[string]interface{}
	Flashes map[string]interface{}
}

// init ensures that both of the underlying maps have been initialized.
func (s *session) init() {
	if s.Data == nil {
		s.Data = make(map[string]interface{})
	}
	if s.Flashes == nil {
		s.Flashes = make(map[string]interface{})
	}
}

// fromReq returns the map of session values from the request. It will
// never return a nil map, instead, the map will be an initialized empty map
// in the case where the session has no data.
func (s *Session) fromReq(r *http.Request) *session {
	// Fastpath: if the context has already been decoded, access the
	// underlying map and return the value associated with the given key.
	v := r.Context().Value(sessionCtxKey)
	if v != nil {
		ss, ok := v.(*session)
		if ok {
			return ss
		}
	}

	cookie, err := r.Cookie(defaultSessionName)
	if err != nil {
		// The only error that can be returned by r.Cookie() is ErrNoCookie,
		// so if the error is not nil, that means that the cookie doesn't
		// exist. When that is the case, the value associated with the given
		// key is guaranteed to be nil, so we return nil.
		ss := &session{}
		ss.init()
		return ss
	}

	ss := &session{}
	if err := s.sc.Decode(defaultSessionName, cookie.Value, s); err != nil {
		log.Println("[error] failed to decode session from cookie:", err)
		ss.init()
		return ss
	}
	return ss
}

// saveCtx saves a map of session data in the current request's context. It
// also updates the Set-Cookie header of the
func (s *Session) saveCtx(w http.ResponseWriter, r *http.Request, session *session) {
	ctx := context.WithValue(r.Context(), sessionCtxKey, session)
	r2 := r.Clone(ctx)
	*r = *r2

	encoded, err := s.sc.Encode(defaultSessionName, session)
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
	return data.Data[key]
}

// List returns all key value pairs of session data from the given request.
func (s *Session) List(r *http.Request) map[string]interface{} {
	return s.fromReq(r).Data
}

// Set sets or updates the given value on the session.
func (s *Session) Set(w http.ResponseWriter, r *http.Request, key string, value interface{}) {
	data := s.fromReq(r)
	data.Data[key] = value
	s.saveCtx(w, r, data)
}

// Delete deletes and returns the session value with the given key.
func (s *Session) Delete(w http.ResponseWriter, r *http.Request, key string) interface{} {
	data := s.fromReq(r)
	value := data.Data[key]
	delete(data.Data, key)
	s.saveCtx(w, r, data)
	return value
}

// Flash sets a flash message on a request.
func (s *Session) Flash(w http.ResponseWriter, r *http.Request, key string, value interface{}) {
	data := s.fromReq(r)
	data.Flashes[key] = value
	s.saveCtx(w, r, data)
}

// Flashes returns all flash messages, clearing all saved flashes.
func (s *Session) Flashes(w http.ResponseWriter, r *http.Request) map[string]interface{} {
	data := s.fromReq(r)

	// Copy the map before clearing it from the session.
	values := make(map[string]interface{})
	for k, v := range data.Flashes {
		values[k] = v
	}

	data.Flashes = make(map[string]interface{})

	s.saveCtx(w, r, data)
	return values
}
