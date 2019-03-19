package controllers

import (
	"bytes"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/go-seatbelt/seatbelt"
	"github.com/go-seatbelt/seatbelt/i18n"
	"github.com/sirupsen/logrus"
)

// Render is used for rendering data.
type Render struct {
	IsDevelopment bool
	Funcs         template.FuncMap

	templates *template.Template
	lock      sync.Mutex

	// Computed options.
	hasLayout bool

	// Stuff I'm not sure about yet.
	Basepath string
}

// NewRender returns a new instance of a template renderer.
func NewRender() *Render {
	render := &Render{}
	render.compile()
	return render
}

// HTML renders an HTML response.
func (render *Render) HTML(w http.ResponseWriter, r *http.Request, view string, data seatbelt.Data) {
	if render.IsDevelopment {
		render.compile()
	}

	// If the renderer hasn't been initialized, do so.
	if render.templates == nil {
		render.compile()
	}

	// Assign a layout if there is one.
	if render.hasLayout {
		render.addLayoutFuncs(view, data)
		view = "layout"
	}

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)

	// Register the i18n func in the template's funcmap.
	if tpl := render.templates.Lookup(view); tpl != nil {
		tpl.Funcs(template.FuncMap{
			"t": func(name string, args ...interface{}) string {
				return i18n.T(r, name, args)
			},
		})
	}

	if err := render.templates.ExecuteTemplate(w, view, data); err != nil {
		logrus.Errorf("Error rendering template: %+v", err)
	}
}

// compile reads all templates from the given directory.
//
// We also compute some internally used properties:
//	* If there's a `layout` file or not.
//
func (render *Render) compile() {
	render.lock.Lock()
	defer render.lock.Unlock()

	var dir string
	if render.Basepath != "" {
		dir = filepath.Join(render.Basepath, "views")
	} else {
		dir = "views"
	}

	render.templates = template.New(dir)

	// Walk the supplied directory and compile any files that match our
	// extension list.
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		// Fix same-extension-dirs bug: some dir might be named to:
		// "users.tmpl", "local.html". These dirs should be excluded as they
		// are not valid golang templates, but files under them should be
		// treated as normal. If it is a dir, return immediately (dir is not a
		// valid golang template).
		if info == nil || info.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}

		ext := ""
		if strings.Index(rel, ".") != -1 {
			ext = filepath.Ext(rel)
		}

		extensions := []string{".html", ".gohtml", ".tmpl"}
		for _, extension := range extensions {
			if ext == extension {
				buf, err := ioutil.ReadFile(path)
				if err != nil {
					panic(err)
				}

				name := (rel[0 : len(rel)-len(ext)])

				// If there's a template with the name "layout", we use that
				// as our layout file.
				if name == "layout" {
					render.hasLayout = true
				}

				// TODO consider if this should copy.
				tmpl := render.templates.New(filepath.ToSlash(name))

				// Add our funcmaps.
				if render.Funcs != nil {
					tmpl.Funcs(render.Funcs)
				}

				// Break out if this parsing fails. We don't want any silent
				// server starts.
				template.Must(tmpl.Funcs(helperFuncs).Parse(string(buf)))
				break
			}
		}
		return nil
	})
}

// Included helper functions for use when rendering HTML.
var helperFuncs = template.FuncMap{
	"yield": func() (string, error) {
		return "", fmt.Errorf("yield called with no layout defined")
	},
	"t": func(name string, args ...interface{}) string {
		return "t called with no translations defined"
	},
}

// addLayoutFuncs adds all library-defined layout funcs, overwriting the
// helpers above.
func (render *Render) addLayoutFuncs(view string, binding interface{}) {
	funcs := template.FuncMap{
		"yield": func() (template.HTML, error) {
			buf := new(bytes.Buffer)
			err := render.templates.ExecuteTemplate(buf, view, binding)
			// Return safe HTML here since we are rendering our own template.
			return template.HTML(buf.String()), err
		},
		"t": func(name string, args ...interface{}) string {
			return "t called with no translations defined"
		},
	}

	if tpl := render.templates.Lookup(view); tpl != nil {
		tpl.Funcs(funcs)
	}
}
