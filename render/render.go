// Package render provides functionality for rendering HTML when building web
// applications.
package render

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/unrolled/render"
)

// A ContextualFuncMap is a function that returns an HTML template.FuncMap.
//
// It is used to provide a renderer with template function that have
// request-scoped data.
type ContextualFuncMap func(w http.ResponseWriter, r *http.Request) template.FuncMap

type Render struct {
	re    *render.Render
	funcs []ContextualFuncMap
}

type Options struct {
	// The directory to serve templates from. Default is "templates".
	Dir string

	// The template to use as a layout. Layouts can call {{ yield }}. Defaults
	// to an empty string (meaning a layout is not used).
	Layout string

	// Request-scoped template funcs. Default is nil.
	Funcs []ContextualFuncMap

	// Whether or not to recompile templates. Default is false. Do not
	// recompile templates in production, as this adds a significant
	// performance penalty.
	Reload bool
}

func New(o *Options) *Render {
	if o == nil {
		o = &Options{}
	}

	// TODO Consider adding a preprocessing step to:
	//
	// 	* Automatically add the required templates suffixes within the
	//	  `define` blocks.
	//	* Allow for a default value for the `{{ partial }}` helper func,
	//	  potentially something that expands
	//		{{ partial "title" "fallback "}}
	//	  to
	//		{{ $title := partial "title" }}
	//		{{ with $title }}{{ . }}{{ else }}fallback{{ end }}
	//	  which is painfully verbose.

	// Mock the template funcs by passing in the user-defined template funcs
	// as no-ops in order for the templates to compile successfully. The real
	// implementations are injected at render time.
	mocks := make(map[string]interface{})
	if o.Funcs != nil {
		for _, fn := range o.Funcs {
			if fn != nil {
				for k := range fn(nil, nil) {
					mocks[k] = func() template.HTML { return "" }
				}
			}
		}
	}

	re := render.New(render.Options{
		Directory:     o.Dir,
		Layout:        o.Layout,
		Extensions:    []string{".html"},
		IsDevelopment: o.Reload,
		Funcs:         []template.FuncMap{mocks},
	})

	return &Render{
		re:    re,
		funcs: o.Funcs,
	}
}

type RenderOptions struct {
	Layout     string
	StatusCode int
	Headers    map[string]string
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

// HTML renders the HTML template with the given name. The HTTP request is
// optional, and can be set to nil. It is only used to add request-specific
// context to HTML template functions.
func (r *Render) HTML(w io.Writer, req *http.Request, name string, data map[string]interface{}, opts ...RenderOptions) {
	var o RenderOptions
	for _, opt := range opts {
		o = opt
	}
	o.setDefaults()

	// Prevent read from nil errors by ensuring the map is always initialized.
	//
	// TODO In Seatbelt, this should come from the newly propsed `Values`
	// feature.
	if data == nil {
		data = make(map[string]interface{})
	}

	// Prepare the render options.
	htmlOpts := render.HTMLOptions{Layout: o.Layout}

	// Add the template funcs, providing the context of the current request,
	// if one is provided.
	rw, ok := w.(http.ResponseWriter)
	if ok {
		if req != nil {
			if r.funcs != nil {
				mergedFuncMap := make(map[string]interface{})

				for _, fn := range r.funcs {
					if fn != nil {
						m := fn(rw, req)
						for k, v := range m {
							if _, ok := mergedFuncMap[k]; ok {
								fmt.Printf("[warning] seatbelt/render.HTML: func %s overrides existing func\n", k)
							} else {
								mergedFuncMap[k] = v
							}
						}
					}
				}

				htmlOpts.Funcs = mergedFuncMap
			}
		}
	}
	if ok {
		for k, v := range o.Headers {
			rw.Header().Set(k, v)
		}
	}

	// Even if the template isn't found, the given status code is respected.
	// This is somewhat confusing, but works better in a
	// Turbo (https://turbo.hotwired.dev/) context because it causes the error
	// page rendered by unrolled/render to actually show  up instead of being
	// silently dropped due to the >=400 level status code.
	//
	// If an error occurs, the reponse has already been written meaning that
	// it's too late to intervene, so the best we can do is log it.
	if err := r.re.HTML(w, o.StatusCode, name, data, htmlOpts); err != nil {
		log.Printf("seatbelt/render: failed to render template: %v", err)
	}
}
