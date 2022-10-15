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

// A ContextualFuncMap is a function that returns an HTML template.FuncMap.
//
// It is used to provide a renderer with template function that have
// request-scoped data.
type ContextualFuncMap func(w http.ResponseWriter, r *http.Request) template.FuncMap

type Render struct {
	dir    string
	mu     sync.Mutex
	pool   sync.Pool
	tpls   map[string]map[string]*template.Template
	funcs  []ContextualFuncMap
	reload bool
}

type Options struct {
	Dir    string
	Funcs  []ContextualFuncMap
	Reload bool
}

func New(o Options) *Render {
	r := &Render{
		dir: o.Dir,
		pool: sync.Pool{
			New: func() interface{} {
				return &bytes.Buffer{}
			},
		},
		funcs:  o.Funcs,
		reload: o.Reload,
	}

	r.mu.Lock()
	if err := r.parseTemplates(); err != nil {
		panic("seatbelt/render: failed to parse templates: " + err.Error())
	}
	r.mu.Unlock()

	return r
}

// extractNameFromPath extracts a template name from a filepath relative to
// the given root directory.
func extractNameFromPath(rootDir, path string, replaceSlash bool) string {
	rel, err := filepath.Rel(rootDir, path)
	if err != nil {
		panic(fmt.Sprintf("seatbelt/render: failed to determine relative dir from %s to %s: %v",
			rootDir, path, err))
	}

	var ext string
	if strings.Contains(rel, ".") {
		ext = filepath.Ext(rel)
	}
	if ext != ".html" {
		panic(fmt.Sprintf("seatbelt/render: template %s must end in .html, got %s\n", path, ext))
	}

	name := rel[0 : len(rel)-len(ext)]

	// On Windows, replace the OS-specific path separator "\" with the
	// conventional Linux/Mac one.
	if replaceSlash {
		name = strings.Replace(name, `\`, "/", -1)
	}
	return name
}

func (r *Render) parseTemplates() error {
	if r.tpls == nil {
		r.tpls = make(map[string]map[string]*template.Template)
	}

	dirfs := os.DirFS(r.dir)

	// Parse the layout template in order to clone it for each template later.
	// Because template functions may be defined in the layout, we also need
	// to add them prior to actually doing any parsing.
	basetpl := template.New("layout")

	// Define an initial map of template funcs as no-ops. This will be passed
	// to templates **before** parsing in order to prevent a "function not
	// defined" error. This func map is redefined when templates are rendered,
	// and is provided with request-specific information.
	if r.funcs != nil {
		noopFuncMap := make(map[string]interface{})
		for _, fn := range r.funcs {
			if fn != nil {
				for name := range fn(nil, nil) {
					noopFuncMap[name] = func() struct{} { return struct{}{} }
				}
			}
		}
		basetpl.Funcs(noopFuncMap)
	}

	// Create a temporary map to associate a name with the layout templates.
	// In order to support Go templates' `block` functionality, we'll need to
	// create a clone of each set of actual templates for each layout template.
	layoutTplMap := make(map[string]*template.Template)

	// Parse all of the layout templates into the top-level of the template
	// map.
	layoutRootDir := filepath.Join(".", "layouts")
	if err := fs.WalkDir(dirfs, layoutRootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d == nil || d.IsDir() {
			return nil
		}

		layoutName := extractNameFromPath(layoutRootDir, path, false)

		layoutTemplate, err := basetpl.ParseFS(dirfs, filepath.Join("layouts", layoutName+".html"))
		if err != nil {
			log.Printf("[error] parsing layout template %s: %v", layoutName, err)
			return err
		}

		clone, err := layoutTemplate.Clone()
		if err != nil {
			return fmt.Errorf("seatbelt/render: failed to clone layout template: %w", err)
		}

		_, err = clone.ParseFS(dirfs, path)
		if err != nil {
			return fmt.Errorf("seatbelt/render: failed to parse layout template %s at path %s: %w", layoutName, path, err)
		}

		layoutTplMap[layoutName] = clone

		return nil
	}); err != nil {
		return err
	}

	rootDir := "."
	return fs.WalkDir(dirfs, rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d == nil || d.IsDir() {
			return nil
		}

		name := extractNameFromPath(rootDir, path, true)
		// TODO Replace usage of `HasPrefix` because it's deprecated.
		if filepath.HasPrefix(name, "layouts") {
			return nil
		}

		for layoutName, layoutTemplate := range layoutTplMap {
			clone, err := layoutTemplate.Clone()
			if err != nil {
				return fmt.Errorf("seatbelt/render: failed to clone layout template when rendering %s: %w", name, err)
			}

			_, err = clone.ParseFS(dirfs, path)
			if err != nil {
				return fmt.Errorf("seatbelt/render: failed to parse template %s at path %s: %w", name, path, err)
			}

			if m, ok := r.tpls[layoutName]; ok {
				m[name] = clone
			} else {
				r.tpls[layoutName] = map[string]*template.Template{
					name: clone,
				}
			}
		}

		return nil
	})
}

type RenderOptions struct {
	StatusCode int
	Layout     string
}

func (r *RenderOptions) setDefaults() {
	if r.StatusCode == 0 {
		r.StatusCode = http.StatusOK
	}
	if r.Layout == "" {
		r.Layout = "layout"
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
	layoutNames := make([]string, 0, len(r.tpls))
	names := make([]string, 0, 8)

	var hasWalkedNonLayoutTpls bool
	for layoutName := range r.tpls {
		layoutNames = append(layoutNames, layoutName)

		if !hasWalkedNonLayoutTpls {
			for nonLayoutName := range r.tpls[layoutName] {
				names = append(names, nonLayoutName)
			}
			hasWalkedNonLayoutTpls = true
		}
	}

	return "layouts: " + strings.Join(layoutNames, ",") +
		", templates: " + strings.Join(names, ",")
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
	layout, ok := r.tpls[o.Layout]
	if !ok {
		msg := fmt.Sprintf(`seatbelt/template: no layout named "%s", defined templates are: %s`,
			name, r.DefinedTemplates())
		r.TextError(w, msg, http.StatusInternalServerError)
		return
	}
	tpl, ok := layout[name]
	if r.reload {
		r.mu.Unlock()
	}
	if !ok {
		msg := fmt.Sprintf(`seatbelt/render: no template named "%s", defined templates are: %s`,
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
	rw, ok := w.(http.ResponseWriter)
	if ok {
		if req != nil {
			if r.funcs != nil {
				for _, fn := range r.funcs {
					if fn != nil {
						tpl.Funcs(fn(rw, req))
					}
				}
			}
		}
	}

	if err := tpl.ExecuteTemplate(b, o.Layout+".html", data); err != nil {
		log.Printf("seatbelt/render: Failed to execute template %s with layout %s and error: %v. Defined templates are: %s",
			name, o.Layout, err, r.DefinedTemplates())
		r.TextError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if ok {
		rw.Header().Set("Content-Type", "text/html")
		rw.WriteHeader(o.StatusCode)
	}
	if _, err := b.WriteTo(w); err != nil {
		log.Println("seatbelt/render: failed to write response:", err)
	}
}
