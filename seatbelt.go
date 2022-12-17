package seatbelt

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-seatbelt/seatbelt/handler"
	"github.com/go-seatbelt/seatbelt/render"
	"github.com/go-seatbelt/seatbelt/session"

	"github.com/go-chi/chi"
	"github.com/gorilla/csrf"
)

// ChiPathParamFunc extracts path parameters from the given HTTP request using
// the github.com/go-chi/chi router.
func ChiPathParamFunc(r *http.Request, values map[string]interface{}) {
	if rctx := chi.RouteContext(r.Context()); rctx != nil {
		for i, key := range rctx.URLParams.Keys {
			values[key] = rctx.URLParams.Values[i]
		}
	}
}

type Context struct {
	r        *http.Request
	w        http.ResponseWriter
	session  *session.Session
	renderer *render.Render
	values   map[string]any
}

func (c *Context) Params(v interface{}) error {
	return handler.Params(c.w, c.r, ChiPathParamFunc, v)
}

func (c *Context) Redirect(url string) error {
	handler.Redirect(c.w, c.r, url)
	return nil
}

// PathParam returns the path param with the given name.
func (c *Context) PathParam(name string) string {
	return chi.URLParam(c.r, name)
}

// FormValue returns the form value with the given name.
func (c *Context) FormValue(name string) string {
	return c.r.FormValue(name)
}

// QueryParam returns the URL query parameter with the given name.
func (c *Context) QueryParam(name string) string {
	return c.r.URL.Query().Get(name)
}

// JSON renders a JSON response with the given status code and data.
func (c *Context) JSON(code int, v interface{}) error {
	return handler.JSON(c.w, code, v)
}

// String sends a string response with the given status code.
func (c *Context) String(code int, s string) error {
	c.w.Header().Set("Content-Type", "text/plain")
	c.w.WriteHeader(code)
	_, err := c.w.Write([]byte(s))
	return err
}

// NoContent sends a 204 No Content HTTP response. It will always return a nil
// error.
func (c *Context) NoContent() error {
	c.w.WriteHeader(204)
	return nil
}

// SetValue sets the given key value pair on the request. These values are
// passed to every HTML template by merging them with the given `data`.
func (c *Context) SetValue(key string, value any) {
	if c.values == nil {
		c.values = make(map[string]any)
	}
	c.values[key] = value
}

// GetValue returns the request-scoped value with the given key.
func (c *Context) GetValue(key string) any {
	if c.values != nil {
		return c.values[key]
	}
	return nil
}

// ListValues returns all request-scoped values.
func (c *Context) ListValues() map[string]any {
	if c.values != nil {
		return c.values
	}
	return nil
}

// DeleteValue deletes the given request-scoped value.
func (c *Context) DeleteValue(key string) {
	if c.values != nil {
		delete(c.values, key)
	}
}

func (c *Context) mergeValues(data map[string]interface{}) map[string]interface{} {
	if c.values == nil {
		return data
	}

	if data == nil {
		return c.values
	}

	for k, v := range c.values {
		if _, ok := data[k]; !ok {
			data[k] = v
		}
	}
	return data
}

// Render renders an HTML template.
//
// If there are any request-scoped values present on the request, they will
// be merged with the given data, with the data taking precendence in case of
// key collisions.
//
// Render will never return an error, and only has the function signature as a
// convenience for writing shorter handlers, for example,
//
//	func ShowNewUser(c *seatbelt.Context) error {
//		return c.Render("users/new", nil)
//	}
func (c *Context) Render(name string, data map[string]interface{}, opts ...render.RenderOptions) error {
	c.renderer.HTML(c.w, c.r, name, c.mergeValues(data), opts...)
	return nil
}

// Set sets or updates the given value on the session.
func (c *Context) Set(key string, value interface{}) {
	c.session.Set(c.w, c.r, key, value)
}

// Get returns the value associated with the given key in the request session.
func (c *Context) Get(key string) interface{} {
	return c.session.Get(c.r, key)
}

// List returns all key value pairs of session data from the given request.
func (c *Context) List() map[string]interface{} {
	return c.session.List(c.r)
}

// Delete deletes the session data with the given key. The deleted session
// data is returned.
func (c *Context) Delete(key string) interface{} {
	return c.session.Delete(c.w, c.r, key)
}

// Reset deletes all values from the session data.
func (c *Context) Reset() {
	c.session.Reset(c.w, c.r)
}

// Flash sets a flash message on a request.
func (c *Context) Flash(key string, value interface{}) {
	c.session.Flash(c.w, c.r, key, value)
}

// Flashes returns all flash messages, clearing all saved flashes.
func (c *Context) Flashes() map[string]interface{} {
	return c.session.Flashes(c.w, c.r)
}

// Request returns the underlying *http.Request belonging to the current
// request context.
func (c *Context) Request() *http.Request {
	return c.r
}

// Response returns the underlying http.ResponseWriter belonging to the
// current request context.
func (c *Context) Response() http.ResponseWriter {
	return c.w
}

// An App contains the data necessary to start and run an application.
//
// An App acts as a router. You must provide your own HTTP server in order to
// start it in the application, i.e.:
//
//	app := seatbelt.New()
//	http.ListenAndServe(":3000", app)
//
// Or,
//
//	app := seatbelt.New()
//	srv := &http.Server{
//		Handler: app,
//	}
//	srv.ListenAndServe()
type App struct {
	// The signing key for the session and CSRF cookies.
	signingKey []byte

	// First party dependencies on the session and render packages.
	session  *session.Session
	renderer *render.Render

	// The HTTP router and its configuration options.
	mux          chi.Router
	middlewares  []MiddlewareFunc
	errorHandler func(c *Context, err error)
}

// MiddlewareFunc is the type alias for Seatbelt middleware.
type MiddlewareFunc func(fn func(ctx *Context) error) func(*Context) error

// An Option is used to configure a Seatbelt application.
type Option struct {
	// The directory containing your HTML templates.
	TemplateDir string

	// The signing key for the session cookie store.
	SigningKey string

	// The session name for the session cookie. Default is "_session".
	SessionName string

	// The MaxAge for the session cookie. Default is 365 days. Pass -1 for no
	// max age.
	SessionMaxAge int

	// Request-contextual HTML functions.
	Funcs func(w http.ResponseWriter, r *http.Request) template.FuncMap

	// Whether or not to reload templates on each request.
	Reload bool

	// SkipServeFiles does not automatically serve static files from the
	// project's /public directory when set to true. Default is false.
	SkipServeFiles bool
}

// setDefaults sets the default values for Seatbelt options.
func (o *Option) setDefaults() {
	if o.TemplateDir == "" {
		o.TemplateDir = "templates"
	}
	if o.SigningKey == "" {
		o.setMasterKey()
	}
}

// setMasterKey makes sure that a master key is set. If the "SECRET"
// environment variable is set, that value is used. If not, we check the
// "master.key" file to see if it exists. If it is, its value is used, and if
// not, a random 64 character hex encoded string is generated and written to
// a new "master.key" file.
//
// The "master.key" file is a secret and should be treated as such. It should
// not be checked into your source code, and in production, the "SECRET"
// environment variable should instead be used.
func (o *Option) setMasterKey() {
	if key := os.Getenv("SECRET"); key != "" {
		o.SigningKey = key
	}

	if key, err := os.ReadFile("master.key"); err == nil {
		o.SigningKey = string(key)
		return
	}

	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("seatbelt: failed to read from source of randomness while generating master.key: %v", err))
	}
	key := make([]byte, hex.EncodedLen(len(b)))
	hex.Encode(key, b)

	if err := os.WriteFile("master.key", key, 0600); err != nil {
		log.Fatalln("seatbelt: failed to write newly generated master.key:", err)
	}
	o.SigningKey = string(key)
}

// defaultTemplateFuncs sets default HTML template functions on each request
// context.
func defaultTemplateFuncs(session *session.Session) func(w http.ResponseWriter, r *http.Request) template.FuncMap {
	return func(w http.ResponseWriter, r *http.Request) template.FuncMap {
		return template.FuncMap{
			"csrf": func() template.HTML {
				return csrf.TemplateField(r)
			},
			"flashes": func() map[string]interface{} {
				return session.Flashes(w, r)
			},
			// versionpath takes a filepath and returns the same filepath with
			// a query parameter appended that contains the unix timestamp of
			// that file's last modified time. This should be used for files
			// that might change between page loads (JavaScript and CSS files,
			// images, etc).
			"versionpath": func(path string) string {
				path = filepath.Clean(path)

				// Leading `/` characters will just break local filepath
				// resolution, so we remove it if it exists.
				fi, err := os.Stat(strings.TrimPrefix(path, "/"))
				if err == nil {
					path = path + "?" + strconv.Itoa(int(fi.ModTime().Unix()))
				} else {
					fmt.Printf("seatbelt: error getting file info at path %s: %v\n", path, err)
				}

				return path
			},
			"csrfMetaTags": func() template.HTML {
				return template.HTML(`<meta name="csrf-token" content="` + csrf.Token(r) + `">`)
			},
		}
	}
}

// New returns a new instance of a Seatbelt application.
func New(opts ...Option) *App {
	var opt Option
	for _, o := range opts {
		opt = o
	}
	opt.setDefaults()

	signingKey, err := hex.DecodeString(opt.SigningKey)
	if err != nil {
		log.Fatalf("seatbelt: signing key is not a valid hexadecimal string: %+v", err)
	}

	// Initialize the underlying chi mux so that we can setup our default
	// middleware stack.
	mux := chi.NewRouter()
	mux.Use(csrf.Protect(signingKey))

	sess := session.New(signingKey, session.Options{
		Name:   opt.SessionName,
		MaxAge: opt.SessionMaxAge,
	})

	funcMaps := []render.ContextualFuncMap{defaultTemplateFuncs(sess)}
	if opt.Funcs != nil {
		funcMaps = append(funcMaps, opt.Funcs)
	}

	app := &App{
		mux:        mux,
		signingKey: signingKey,
		session:    sess,
		renderer: render.New(&render.Options{
			Dir:    opt.TemplateDir,
			Layout: "layout",
			Reload: opt.Reload,
			Funcs:  funcMaps,
		}),
	}

	if !opt.SkipServeFiles {
		app.FileServer("/public", "public")
	}

	return app
}

// Start is a convenience method for starting the application server with a
// default *http.Server.
//
// Start should not be used in production, as the standard library's default
// HTTP server is not suitable for production use due to a lack of timeouts,
// etc.
//
// Production applications should create their own
// *http.Server, and pass the *seatbelt.App to that *http.Server's `Handler`.
func (a *App) Start(addr string) error {
	return http.ListenAndServe(addr, a)
}

// UseStd registers standard HTTP middleware on the application.
func (a *App) UseStd(middleware ...func(http.Handler) http.Handler) {
	a.mux.Use(middleware...)
}

// Use registers Seatbelt HTTP middleware on the application.
func (a *App) Use(middleware ...MiddlewareFunc) {
	a.middlewares = append(a.middlewares, middleware...)
}

// SetErrorHandler allows you to set a custom error handler that runs when an
// error is returned from an HTTP handler.
func (a *App) SetErrorHandler(fn func(c *Context, err error)) {
	a.errorHandler = fn
}

// ErrorHandler is the globally registered error handler.
//
// You can override this function using `SetErrorHandler`.
func (a *App) handleErr(c *Context, err error) {
	if a.errorHandler != nil {
		a.errorHandler(c, err)
		return
	}

	fmt.Printf("seatbelt: hit error handler: %#v\n", err)

	switch c.r.Method {
	case "GET", "HEAD", "OPTIONS":
		c.String(http.StatusInternalServerError, err.Error())
	default:
		from := c.r.Referer()
		c.Flash("alert", err.Error())
		c.Redirect(from)
	}
}

// serveContext creates and registers a Seatbelt handler for an HTTP request.
func (a *App) serveContext(w http.ResponseWriter, r *http.Request, handle func(c *Context) error) {
	c := &Context{w: w, r: r, session: a.session, renderer: a.renderer}

	// Iterate over the middleware in reverse order, so that the order
	// in which middleware is registered suggests that it is run from
	// the outermost (or leftmost) function to the innermost (or
	// rightmost) function.
	//
	// This means if you register two middlewares like,
	//	app.Use(m1, m2)
	// It will run as:
	//	m1->m2->handler->m2 returned->m1 returned.
	for i := len(a.middlewares) - 1; i >= 0; i-- {
		handle = a.middlewares[i](handle)
	}

	if err := handle(c); err != nil {
		a.handleErr(c, err)
	}
}

// handle registers the given handler to handle requests at the given path
// with the given HTTP verb.
func (a *App) handle(verb, path string, handle func(c *Context) error) {
	switch verb {
	case "HEAD":
		a.mux.Head(path, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			a.serveContext(w, r, handle)
		}))

	case "OPTIONS":
		a.mux.Options(path, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			a.serveContext(w, r, handle)
		}))

	case "GET":
		a.mux.Get(path, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			a.serveContext(w, r, handle)
		}))

	case "POST":
		a.mux.Post(path, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			a.serveContext(w, r, handle)
		}))

	case "PUT":
		a.mux.Put(path, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			a.serveContext(w, r, handle)
		}))

	case "PATCH":
		a.mux.Patch(path, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			a.serveContext(w, r, handle)
		}))

	case "DELETE":
		a.mux.Delete(path, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			a.serveContext(w, r, handle)
		}))

	default:
		panic("method " + verb + " not allowed")
	}
}

// Head routes HEAD requests to the given path.
func (a *App) Head(path string, handle func(c *Context) error) {
	a.handle("HEAD", path, handle)
}

// Options routes OPTIONS requests to the given path.
func (a *App) Options(path string, handle func(c *Context) error) {
	a.handle("OPTIONS", path, handle)
}

// Get routes GET requests to the given path.
func (a *App) Get(path string, handle func(c *Context) error) {
	a.handle("GET", path, handle)
}

// Post routes POST requests to the given path.
func (a *App) Post(path string, handle func(c *Context) error) {
	a.handle("POST", path, handle)
}

// Put routes PUT requests to the given path.
func (a *App) Put(path string, handle func(c *Context) error) {
	a.handle("PUT", path, handle)
}

// Patch routes PATCH requests to the given path.
func (a *App) Patch(path string, handle func(c *Context) error) {
	a.handle("PATCH", path, handle)
}

// Delete routes DELETE requests to the given path.
func (a *App) Delete(path string, handle func(c *Context) error) {
	a.handle("DELETE", path, handle)
}

// FileServer serves the contents of the given directory at the given path.
func (a *App) FileServer(path string, dir string) {
	if strings.ContainsAny(path, "{}*") {
		panic("FileServer does not permit URL parameters.")
	}

	fs := http.StripPrefix(path, http.FileServer(http.Dir(dir)))

	if path != "/" && path[len(path)-1] != '/' {
		a.mux.Get(path, http.RedirectHandler(path+"/", http.StatusMovedPermanently).ServeHTTP)
		path += "/"
	}
	path += "*"

	a.mux.Get(path, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fs.ServeHTTP(w, r)
	}))
}

// ServeHTTP makes the Seatbelt application implement the http.Handler
// interface.
func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.mux.ServeHTTP(w, r)
}
