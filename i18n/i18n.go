package i18n

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

type Translator struct {
	path          string
	bundle        *i18n.Bundle
	isDevelopment bool
}

// New creates a new instance of a translator from the given file path.
func New(path string, isDevelopment bool) *Translator {
	bundle := i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)

	translator := &Translator{
		path:          path,
		bundle:        bundle,
		isDevelopment: isDevelopment,
	}
	translator.parseTranslationFiles()

	return translator
}

func (t *Translator) parseTranslationFiles() {
	// If the path is an empty string, we'll fall back to the default bundle,
	// which will output the "translation missing" error for every string.
	// Otherwise, load the translation data from the given filepath.
	if t.path != "" {
		if err := filepath.Walk(t.path, func(filepath string, info os.FileInfo, _ error) error {
			if info == nil || info.IsDir() {
				return nil
			}
			// TODO Check file extension (or possibly regex of filename) so
			// that it doesn't break on unintenionally added files.
			_, err := t.bundle.LoadMessageFile(filepath)
			return err
		}); err != nil {
			panic(err)
		}
	}
}

// T translates the string with the given name.
func (t *Translator) T(r *http.Request, id string, data map[string]interface{}, pluralCount ...int) string {
	lang := r.URL.Query().Get("locale")
	accept := r.Header.Get("Accept-Language")

	if t.isDevelopment {
		t.parseTranslationFiles()
	}

	localizer := i18n.NewLocalizer(t.bundle, lang, accept)

	lc := &i18n.LocalizeConfig{
		MessageID:    id,
		TemplateData: data,
	}
	for _, pc := range pluralCount {
		lc.PluralCount = pc
	}

	text, err := localizer.Localize(lc)
	if err != nil {
		return "translation missing: " + guessLang(accept, lang) + ", " + id
	}

	return text
}

// TODO Make this work the exact same as i18n.NewLocalizer
func guessLang(langs ...string) string {
	defaultLang := language.English.String()

	if len(langs) == 0 {
		return defaultLang
	}

	var guessedLang string
	for i := len(langs) - 1; i != 0; i-- {
		if langs[i] == "" {
			continue
		}

		tag, err := language.Parse(langs[i])
		if err != nil {
			continue
		}
		guessedLang = tag.String()
	}

	if guessedLang == "" {
		return defaultLang
	}
	return guessedLang
}
