package controllers

import (
	"html/template"
	"strings"
)

type Form struct {
	model interface{}
}

func FormFor(model interface{}) *Form {
	return &Form{
		model: model,
	}
}

func (f *Form) TextField(s string) template.HTML {
	b := strings.Builder{}

	b.WriteString(`  <input type="text" name="`)
	b.WriteString(s)
	b.WriteString(`"/>`)

	return template.HTML(b.String())
}

func (f *Form) Submit(s string) template.HTML {
	b := strings.Builder{}

	b.WriteString(`  <input type="submit" value="`)
	b.WriteString(s)
	b.WriteString(`"/>`)

	return template.HTML(b.String())
}
