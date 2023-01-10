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
	"github.com/go-seatbelt/seatbelt/i18n"
	"github.com/go-seatbelt/seatbelt/render"
	"github.com/go-seatbelt/seatbelt/session"
	"github.com/go-seatbelt/seatbelt/values"

	"github.com/go-chi/chi"
	"github.com/gorilla/csrf"
)

// Version is the version of the Seatbelt package.
const Version = "v0.2.0"

// ChiPathParamFunc extracts path parameters from the given HTTP request using
// the github.com/go-chi/chi router.
func ChiPathParamFunc(r *http.Request, values map[string]interface{}) {
	if rctx := chi.RouteContext(r.Context()); rctx != nil {
		for i, key := range rctx.URLParams.Keys {
			values[key] = rctx.URLParams.Values[i]
		}
	}
}

type context struct {
	r        *http.Request
	w        http.ResponseWriter
	i18n     *i18n.Translator
	values   *values.Values
	session  *session.Session
	renderer *render.Render
}

type ContextI18N context

func (c *ContextI18N) T(id string, data map[string]any, count ...int) string {
	return c.i18n.T(c.r, id, mergeMaps(c.values.List(), data), count...)
}

type ContextValues context

// Set sets the given key value pair on the request. These values are
// passed to every HTML template by merging them with the given `data`.
func (c *ContextValues) Set(key string, value any) {
	c.values.Set(key, value)
}

// Get returns the request-scoped value with the given key.
func (c *ContextValues) Get(key string) any {
	return c.values.Get(key)
}

// List returns all request-scoped values.
func (c *ContextValues) List() map[string]any {
	return c.values.List()
}

// Delete deletes the given request-scoped value.
func (c *ContextValues) Delete(key string) {
	c.values.Delete(key)
}

type ContextSession context

// Set sets or updates the given value on the session.
func (c *ContextSession) Set(key string, value interface{}) {
	c.session.Set(c.w, c.r, key, value)
}

// Get returns the value associated with the given key in the request session.
func (c *ContextSession) Get(key string) interface{} {
	return c.session.Get(c.r, key)
}

// List returns all key value pairs of session data from the given request.
func (c *ContextSession) List() map[string]interface{} {
	return c.session.List(c.r)
}

// Delete deletes the session data with the given key. The deleted session
// data is returned.
func (c *ContextSession) Delete(key string) interface{} {
	return c.session.Delete(c.w, c.r, key)
}

// Reset deletes all values from the session data.
func (c *ContextSession) Reset() {
	c.session.Reset(c.w, c.r)
}

type ContextFlash context

// Flash adds a flash message on a request.
func (c *ContextFlash) Add(key string, value interface{}) {
	c.session.Flash(c.w, c.r, key, value)
}

// List returns all flash messages, clearing all saved flashes.
func (c *ContextFlash) List() map[string]interface{} {
	return c.session.Flashes(c.w, c.r)
}

func (c *context) Params(v interface{}) error {
	return handler.Params(c.w, c.r, ChiPathParamFunc, v)
}

func (c *context) Redirect(url string) error {
	handler.Redirect(c.w, c.r, url)
	return nil
}

// PathParam returns the path param with the given name.
func (c *context) PathParam(name string) string {
	return chi.URLParam(c.r, name)
}

// FormValue returns the form value with the given name.
func (c *context) FormValue(name string) string {
	return c.r.FormValue(name)
}

// QueryParam returns the URL query parameter with the given name.
func (c *context) QueryParam(name string) string {
	return c.r.URL.Query().Get(name)
}

// JSON renders a JSON response with the given status code and data.
func (c *context) JSON(code int, v interface{}) error {
	return handler.JSON(c.w, code, v)
}

// String sends a string response with the given status code.
func (c *context) String(code int, s string) error {
	c.w.Header().Set("Content-Type", "text/plain")
	c.w.WriteHeader(code)
	_, err := c.w.Write([]byte(s))
	return err
}

// NoContent sends a 204 No Content HTTP response. It will always return a nil
// error.
func (c *context) NoContent() error {
	c.w.WriteHeader(204)
	return nil
}

// GetIP attempts to return the request's IP address, first by checking the
// `X-Real-Ip` header, then the `X-Forwarded-For` header, and finally falling
// back to the request's `RemoteAddr`.
func (c *context) GetIP() string {
	if ip := c.r.Header.Get("X-Real-Ip"); ip != "" {
		return ip
	}
	if ip := c.r.Header.Get("X-Forwarded-For"); ip != "" {
		return ip
	}
	return c.r.RemoteAddr
}

// mergeMaps merges the values of m2 into m1. If a value in m2 has the same
// key as in m1, the key in m1 takes precedence.
func mergeMaps(m1, m2 map[string]interface{}) map[string]interface{} {
	if m1 == nil {
		return m2
	}
	if m2 == nil {
		return m1
	}

	for k, v := range m2 {
		if _, ok := m1[k]; !ok {
			m1[k] = v
		}
	}

	return m1
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
func (c *context) Render(name string, data map[string]interface{}, opts ...render.RenderOptions) error {
	c.renderer.HTML(c.w, c.r, name, mergeMaps(c.values.List(), data), opts...)
	return nil
}

// Request returns the underlying *http.Request belonging to the current
// request context.
func (c *context) Request() *http.Request {
	return c.r
}

// Response returns the underlying http.ResponseWriter belonging to the
// current request context.
func (c *context) Response() http.ResponseWriter {
	return c.w
}

type Context struct {
	context
	I18N    *ContextI18N
	Flash   *ContextFlash
	Values  *ContextValues
	Session *ContextSession
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

	// First party dependencies on the session, render, and i18n packages.
	i18n     *i18n.Translator
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

	// The directory containing your i18n data.
	LocaleDir string

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
func defaultTemplateFuncs(session *session.Session, translator *i18n.Translator) func(w http.ResponseWriter, r *http.Request) template.FuncMap {
	return func(w http.ResponseWriter, r *http.Request) template.FuncMap {
		return template.FuncMap{
			"t": func(id string, data map[string]interface{}, pluralCount ...int) string {
				vals := values.New(r).List()
				return translator.T(r, id, mergeMaps(vals, data), pluralCount...)
			},
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

	translator := i18n.New(opt.LocaleDir, opt.Reload)

	// Initialize the underlying chi mux so that we can setup our default
	// middleware stack.
	mux := chi.NewRouter()
	mux.Use(csrf.Protect(signingKey))

	sess := session.New(signingKey, session.Options{
		Name:   opt.SessionName,
		MaxAge: opt.SessionMaxAge,
	})

	funcMaps := []render.ContextualFuncMap{defaultTemplateFuncs(sess, translator)}
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
		i18n: translator,
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
		c.Flash.Add("alert", err.Error())
		c.Redirect(from)
	}
}

// serveContext creates and registers a Seatbelt handler for an HTTP request.
func (a *App) serveContext(w http.ResponseWriter, r *http.Request, handle func(c *Context) error) {
	common := &context{
		w:        w,
		r:        r,
		i18n:     a.i18n,
		values:   values.New(r),
		session:  a.session,
		renderer: a.renderer,
	}

	c := &Context{
		context: *common,
		I18N:    (*ContextI18N)(common),
		Flash:   (*ContextFlash)(common),
		Values:  (*ContextValues)(common),
		Session: (*ContextSession)(common),
	}

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

// Namespace creates a new *seatbelt.App with an empty middleware stack and
// mounts it on the `pattern` as a subrouter.
func (a *App) Namespace(pattern string, fn func(app *App)) *App {
	if fn == nil {
		panic(fmt.Sprintf("seatbelt: attempting to Route() a nil sub-app on '%s'", pattern))
	}

	subApp := &App{
		signingKey:   a.signingKey,
		i18n:         a.i18n,
		session:      a.session,
		renderer:     a.renderer,
		errorHandler: a.errorHandler,
		mux:          chi.NewRouter(),
		// TODO Not sure if this is actually the behaviour we want -- should
		// it inherit the middleware stack?
		middlewares: make([]MiddlewareFunc, 0),
	}

	fn(subApp)
	a.mux.Mount(pattern, subApp)

	return subApp
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
