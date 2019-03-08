package server

import (
	"bytes"
	"html/template"
	"net/http"
	"strings"
)

const (
	// TurbolinksReferrer is the header sent by the Turbolinks frontend on any
	// XHR requests powered by Turbolinks. We use this header to detect if the
	// current request was sent from Turbolinks.
	TurbolinksReferrer = "Turbolinks-Referrer"

	// TurbolinksCookie is the name of the cookie that we use to handle
	// redirect requests correctly.
	//
	// We name it `_turbolinks_location` to be consistent with the name Rails
	// give to the cookie that serves the same purpose.
	TurbolinksCookie = "_turbolinks_location"
)

// Turbolinks wraps an HTTP handler to support the behaviour required by the
// Turbolinks JavaScript library.
func Turbolinks(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		referer := r.Header.Get(TurbolinksReferrer)
		if referer == "" {
			// Turbolinks isn't enabled, so don't do anything extra.
			h.ServeHTTP(w, r)
			return
		}

		// Check for POST request. If we do encounter a POST request, execute
		// the HTTP handler, but then tell the client to redirect accoringly.
		if r.Method == http.MethodPost {
			rs := &responseStaller{
				w:    w,
				code: 0,
				buf:  &bytes.Buffer{},
			}
			h.ServeHTTP(rs, r)

			// TODO(ben) This opens you up to JavaScript injection via the
			// value of `location`!!
			if location := rs.Header().Get("Location"); location != "" {
				rs.Header().Set("Content-Type", "text/javascript")
				rs.Header().Set("X-Content-Type-Options", "nosniff")
				rs.WriteHeader(http.StatusOK)

				// Remove Location header since we're returning a 200
				// response.
				rs.Header().Del("Location")

				// Create the JavaScript to send to the frontend for
				// redirection after a form submit.
				//
				// Also, escape the location value so that it can't be used
				// for frontend JavaScript injection.
				js := []byte(`Turbolinks.clearCache();Turbolinks.visit("` + template.JSEscapeString(location) + `", {action: "advance"});`)

				// Write the hash of the JavaScript so we can send it in the
				// Content Security Policy header, in order to prevent inline
				// scripts.
				//
				// hash := sha256.New()
				// hash.Write(js)
				// sha := hex.EncodeToString(hash.Sum(nil))
				// rs.Header().Set("Content-Security-Policy", "script-src 'sha256-"+sha+"'")

				rs.Write(js)
			}

			rs.SendResponse()
			return
		}

		// If the Turbolinks cookie is found, then redirect to the location
		// specified in the cookie.
		if cookie, err := r.Cookie(TurbolinksCookie); err == nil {
			w.Header().Set("Turbolinks-Location", cookie.Value)
			cookie.MaxAge = -1
			http.SetCookie(w, cookie)
		}

		// Handle the request. We use a "response staller" here so that,
		//
		//	* The request isn't sent when the underlying http.ResponseWriter
		//	  calls write.
		//	* We can still write to the header after the request is handled.
		//
		// This is done in order to append the `_turbolinks_location` cookie
		// for the requests that need it.
		rs := &responseStaller{
			w:    w,
			code: 0,
			buf:  &bytes.Buffer{},
		}
		h.ServeHTTP(rs, r)

		// Check if a redirect was performed. Is there was, then we need a way
		// to tell the next request to set the special Turbolinks header that
		// will force Turbolinks to update the URL (as push state history) for
		// that redirect. We do this by setting a cookie on this request that
		// we can check on the next request.
		if location := rs.Header().Get("Location"); location != "" {
			http.SetCookie(rs, &http.Cookie{
				Name:     TurbolinksCookie,
				Value:    location,
				Path:     "/",
				HttpOnly: true,
				Secure:   IsTLS(r),
			})
		}

		rs.SendResponse()
	})
}

type responseStaller struct {
	w    http.ResponseWriter
	code int
	buf  *bytes.Buffer
}

// Write is a wrapper that calls the underlying response writer's Write
// method, but write the response to a buffer instead.
func (rw *responseStaller) Write(b []byte) (int, error) {
	return rw.buf.Write(b)
}

// WriteHeader saves the status code, to be sent later during the SendReponse
// call.
func (rw *responseStaller) WriteHeader(code int) {
	rw.code = code
}

// Header wraps the underlying response writers Header method.
func (rw *responseStaller) Header() http.Header {
	return rw.w.Header()
}

// SendResponse writes the header to the underlying response writer, and
// writes the response.
func (rw *responseStaller) SendResponse() {
	rw.w.WriteHeader(rw.code)
	rw.buf.WriteTo(rw.w)
}

// IsTLS is a helper to check if a requets was performed over HTTPS.
func IsTLS(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	if strings.ToLower(r.Header.Get("X-Forwarded-Proto")) == "https" {
		return true
	}
	return false
}
