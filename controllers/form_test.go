package controllers_test

import (
	"bytes"
	"html/template"
	"testing"

	"github.com/go-seatbelt/seatbelt/controllers"
)

const testTpl = `
<form method="POST" action="/posts">
  {{ with $f := form_for . }}
    {{ $f.TextField "title" }}
    {{ $f.Submit "Submit" }}
  {{ end }}
</form>
`

func TestControllers(t *testing.T) {
	t.Parallel()

	tpl := template.Must(template.New("test").Funcs(template.FuncMap{
		"form_for": controllers.FormFor,
	}).Parse(testTpl))

	buf := &bytes.Buffer{}
	if err := tpl.Execute(buf, map[string]interface{}{
		"Model": nil,
	}); err != nil {
		t.Fatalf("error executing template: %+v", err)
	}
}
