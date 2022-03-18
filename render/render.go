// Package render provides functionality for rendering HTML when building web
// applications.
package render

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

type Render struct {
	dir    string
	mu     sync.Mutex
	tpls   map[string]*template.Template
	pool   sync.Pool
	reload bool
	funcs  func(r *http.Request) template.FuncMap
}

type Options struct {
	Dir    string
	Reload bool
	Funcs  func(r *http.Request) template.FuncMap
}

func New(o Options) *Render {
	r := &Render{
		dir: o.Dir,
		pool: sync.Pool{
			New: func() interface{} {
				return &bytes.Buffer{}
			},
		},
		reload: o.Reload,
		funcs:  o.Funcs,
	}

	r.mu.Lock()
	if err := r.parseTemplates(); err != nil {
		log.Fatalln("[fatal] failed to parse templates:", err)
	}
	r.mu.Unlock()

	return r
}

func (r *Render) parseTemplates() error {
	if r.tpls == nil {
		r.tpls = make(map[string]*template.Template)
	}

	dirfs := os.DirFS(r.dir)

	// Parse the layout template in order to clone it for each template later.
	ltpl, err := template.New("layout").ParseFS(dirfs, "layout.html")
	if err != nil {
		log.Fatalln("[error] parsing layout template", err)
		return err
	}

	// Define an initial map of template funcs as no-ops. This will be passed
	// to templates **before** parsing in order to prevent a "function not
	// defined" error. This func map is redefined when templates are rendered,
	// and is provided with request-specific information.
	if r.funcs != nil {
		noopFuncMap := make(map[string]interface{})
		for name := range r.funcs(nil) {
			noopFuncMap[name] = func() struct{} { return struct{}{} }
		}
		ltpl.Funcs(noopFuncMap)
	}

	rootDir := "."
	return fs.WalkDir(dirfs, rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d == nil || d.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(rootDir, path)
		if err != nil {
			return err
		}

		ext := ""
		if strings.Contains(rel, ".") {
			ext = filepath.Ext(rel)
		}
		if ext != ".html" {
			log.Fatalf("[error] template %s must end in .html, got %s\n", path, ext)
			return fmt.Errorf("template %s must end in .html, got %s", path, ext)
		}

		name := rel[0 : len(rel)-len(ext)]

		// On Windows, replace the OS-specific path separator "\" with the
		// conventional Linux/Mac one.
		name = strings.Replace(name, `\`, "/", -1)

		clone, err := ltpl.Clone()
		if err != nil {
			log.Println("[info] failed to clone layout tempalate", err)
			return err
		}

		_, err = clone.ParseFS(dirfs, path)
		if err != nil {
			log.Fatalf("[error] failed to parse template %s at path %s: %#v", name, path, err)
			return fmt.Errorf("failed to parse template %s at path %s: %w", name, path, err)
		}

		r.tpls[name] = clone

		return nil
	})
}

type RenderOptions struct {
	StatusCode int
}

func (r *RenderOptions) setDefaults() {
	if r.StatusCode == 0 {
		r.StatusCode = http.StatusOK
	}
}

// TextError writes the given error message as a plain text response.
func (r *Render) TextError(w io.Writer, error string, code int) {
	if rw, ok := w.(http.ResponseWriter); ok {
		rw.Header().Set("Content-Type", "text/plain; charset=utf-8")
		rw.Header().Set("X-Content-Type-Options", "nosniff")
		rw.Header().Set("Content-Length", strconv.Itoa(len(error)))
		rw.WriteHeader(code)
	}
	w.Write([]byte(error))
}

// DefinedTemplates returns a comma-separated string containing the names of
// all defined templates. Used to generate an error message, but can also be
// used for debugging.
func (r *Render) DefinedTemplates() string {
	names := make([]string, 0, len(r.tpls))
	for name := range r.tpls {
		names = append(names, name)
	}
	return strings.Join(names, ",")
}

// HTML renders the HTML template with the given name. The HTTP request is
// optional, and can be set to nil. It is only used to add request-specific
// context to HTML template functions.
func (r *Render) HTML(w io.Writer, req *http.Request, name string, data map[string]interface{}, opts ...RenderOptions) {
	var o RenderOptions
	for _, opt := range opts {
		o = opt
	}
	o.setDefaults()

	b := r.pool.Get().(*bytes.Buffer)
	b.Reset()
	defer r.pool.Put(b)

	if r.reload {
		r.mu.Lock()
		r.parseTemplates()
		r.mu.Unlock()
	}

	if r.reload {
		r.mu.Lock()
	}
	tpl, ok := r.tpls[name]
	if r.reload {
		r.mu.Unlock()
	}

	if !ok {
		msg := fmt.Sprintf(`no template named "%s", defined templates are %s`,
			name, r.DefinedTemplates())
		r.TextError(w, msg, http.StatusInternalServerError)
		return
	}

	// Prevent read from nil errors by ensuring the map is always initialized.
	if data == nil {
		data = make(map[string]interface{})
	}

	// Add the template funcs, providing the context of the current request,
	// if one is provided.
	if req != nil {
		if r.funcs != nil {
			tpl.Funcs(r.funcs(req))
		}
	}

	if err := tpl.ExecuteTemplate(b, "layout.html", data); err != nil {
		log.Printf("[error] failed to execute template %s: %v", name, err)
		r.TextError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if rw, ok := w.(http.ResponseWriter); ok {
		rw.Header().Set("Content-Type", "text/html")
		rw.WriteHeader(o.StatusCode)
	}
	if _, err := b.WriteTo(w); err != nil {
		log.Println("[error] failed to write response:", err)
	}
}
