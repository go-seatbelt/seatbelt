package controllers

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-seatbelt/seatbelt"
	"github.com/go-seatbelt/seatbelt/internal/config"
	"github.com/go-seatbelt/seatbelt/internal/trace"
	"github.com/sirupsen/logrus"
)

// DefaultController is the default instance of a controller.
var DefaultController *Controller

func init() {
	controller := &Controller{
		Render: &Render{
			Basepath: config.RootPath,
		},
	}

	DefaultController = controller
}

// A Controller handles reading HTTP requests and writing HTTP responses.
type Controller struct {
	*Render
}

// New returns a new instance of a controller.
func New() *Controller {
	return &Controller{NewRender()}
}

// HTML renders the given template.
func HTML(w http.ResponseWriter, r *http.Request, view string, data seatbelt.Data) {
	DefaultController.Render.HTML(w, r, view, data)
}

// JSON writes the given response as JSON. It will respond with 200 OK, unless
// an option status code is provided.
func (c *Controller) JSON(w http.ResponseWriter, r *http.Request, v interface{}, status ...int) {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		c.JSONError(w, "Failed to encode JSON", err, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(buf.Len()))
	for _, s := range status {
		w.WriteHeader(s)
	}

	if _, err := buf.WriteTo(w); err != nil {
		logrus.Errorf("Error writing JSON response: %+v", err)
	}
}

// JSON writes the given response as JSON. It will respond with 200 OK, unless
// an option status code is provided.
func JSON(w http.ResponseWriter, r *http.Request, v interface{}, status ...int) {
	DefaultController.JSON(w, r, v, status...)
}

// JSONError sends the error message as JSON with the given HTTP status code.
func (c *Controller) JSONError(w http.ResponseWriter, message string, err error, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	var errorMsg string
	switch e := err.(type) {
	case *json.SyntaxError:
		errorMsg = err.Error() + " at character " + strconv.Itoa(int(e.Offset))

	default:
		errorMsg = err.Error()
	}

	body, err := json.MarshalIndent(map[string]interface{}{
		"message": message,
		"error":   errorMsg,
	}, "", "  ")
	if err != nil {
		logrus.Errorf("%s: Failed to send JSON response: %+v", trace.Getfl(), err)
	}

	w.Write(body)
}

// JSONError sends the error message as JSON with the given HTTP status code.
func JSONError(w http.ResponseWriter, message string, err error, status int) {
	DefaultController.JSONError(w, message, err, status)
}

const flashCookieName = "_seatbelt_flash"

// Flash sets the given flash message via a cookie on the given response.
func (c *Controller) Flash(w http.ResponseWriter, r *http.Request, flash seatbelt.Flash) {
	buf := &bytes.Buffer{}
	if err := gob.NewEncoder(buf).Encode(flash); err != nil {
		logrus.Errorf("%s: Error encoding flash: %+v", trace.Getfl(), err)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     flashCookieName,
		Value:    base64.StdEncoding.EncodeToString(buf.Bytes()),
		Path:     "/",
		SameSite: http.SameSiteStrictMode,
		HttpOnly: false,
		Secure:   IsSecure(r),
	})
}

// Flash sets the given flash message via a cookie on the given response.
func Flash(w http.ResponseWriter, r *http.Request, flash seatbelt.Flash) {
	DefaultController.Flash(w, r, flash)
}

// GetFlash returns the flash message on the current request and deletes it.
func (c *Controller) GetFlash(w http.ResponseWriter, r *http.Request) seatbelt.Flash {
	cookie, err := r.Cookie(flashCookieName)
	if err != nil {
		return nil
	}

	encodedFlash, err := base64.StdEncoding.DecodeString(cookie.Value)
	if err != nil {
		logrus.Errorf("%s: Error decoding base64 flash: %+v", trace.Getfl(), err)
		return nil
	}

	flash := make(seatbelt.Flash)
	if err := gob.NewDecoder(bytes.NewReader(encodedFlash)).Decode(&flash); err != nil {
		logrus.Errorf("%s: Error decoding flash: %+v", trace.Getfl(), err)
		return nil
	}

	// Delete the session cookie, since the user has seen the flash.
	http.SetCookie(w, &http.Cookie{
		Name:     flashCookieName,
		Value:    "",
		MaxAge:   -1,
		Expires:  time.Unix(1, 0),
		Path:     "/",
		SameSite: http.SameSiteStrictMode,
		HttpOnly: false,
		Secure:   IsSecure(r),
	})
	return flash
}

// GetFlash returns the flash message on the current request and deletes it.
func GetFlash(w http.ResponseWriter, r *http.Request) seatbelt.Flash {
	return DefaultController.GetFlash(w, r)
}

// Redirect is a temporary redirect to the given path. This redirect will always
// respond with http.StatusFound (302). If any flash messages are given,
// they'll be added to the response as a single flash message.
func (c *Controller) Redirect(w http.ResponseWriter, r *http.Request, path string, flash seatbelt.Flash) {
	c.Flash(w, r, flash)
	http.Redirect(w, r, path, http.StatusFound)
}

// Redirect is a temporary redirect to the given path. This redirect will always
// respond with http.StatusFound (302). If any flash messages are given,
// they'll be added to the response as a single flash message.
func Redirect(w http.ResponseWriter, r *http.Request, path string, flash seatbelt.Flash) {
	DefaultController.Redirect(w, r, path, flash)
}

// NotFound is executed anytime the server returns a 404 Not Found response.
func (c *Controller) NotFound(w http.ResponseWriter, r *http.Request) {
	c.HTML(w, r, "404", seatbelt.Data{
		"Path": r.RequestURI,
	})
}

// NotFound is executed anytime the server returns a 404 Not Found response.
func NotFound(w http.ResponseWriter, r *http.Request) {
	DefaultController.NotFound(w, r)
}

// IsSecure is a helper to check if a request was performed over HTTPS.
func IsSecure(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	if strings.ToLower(r.Header.Get("X-Forwarded-Proto")) == "https" {
		return true
	}
	return false
}
