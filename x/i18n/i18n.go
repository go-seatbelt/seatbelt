package i18n

import (
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/sirupsen/logrus"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v2"
)

var bundle *i18n.Bundle

const (
	// I18NCookie is the name of the cookie that sets the user's language
	// preference.
	I18NCookie = "_language"
)

func init() {
	bundle = i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("yml", yaml.Unmarshal)

	localePath := filepath.Join(".", "config", "locales")
	if err := filepath.Walk(localePath, func(path string, info os.FileInfo, _ error) error {
		if info == nil || info.IsDir() {
			return nil
		}
		_, err := bundle.LoadMessageFile(path)
		return err
	}); err != nil {
		panic(err)
	}
}

// T translates the string with the given name.
func T(r *http.Request, name string, args ...interface{}) string {
	var lang string
	cookie, err := r.Cookie(I18NCookie)
	if err == nil {
		lang = cookie.Value
	}

	accept := r.Header.Get("Accept-Language")

	localizer := i18n.NewLocalizer(bundle, lang, accept)
	localizerConfig := &i18n.LocalizeConfig{MessageID: name}

	if len(args) > 1 {
		localizerConfig.PluralCount = args[0]

		data := make(map[string]interface{})
		data["PluralCount"] = localizerConfig.PluralCount

		for i := 1; i < len(args); i++ {
			data["P"+strconv.Itoa(i)] = args[i]
		}
		localizerConfig.TemplateData = data
	}

	text, err := localizer.Localize(localizerConfig)
	if err != nil {
		logrus.Warnf("The string with translation key \"%s\" could not be localized due to error %+v", name, err)
		return name
	}
	return text
}

// SaveLanguage handles the POST request that updates the user's language
// cookie.
func SaveLanguage(w http.ResponseWriter, r *http.Request) {
	referrer, _ := url.Parse(r.Referer())
	if referrer == nil {
		referrer = &url.URL{Path: "/"}
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form body: "+err.Error(), http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     I18NCookie,
		Value:    r.FormValue("language"),
		Path:     "/",
		SameSite: http.SameSiteStrictMode,
		HttpOnly: true,
		Secure:   isSecure(r),
	})
	http.Redirect(w, r, referrer.Path, http.StatusFound)
}

// copied from controllers/controllers.go to avoid an import cycle.
func isSecure(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	if strings.ToLower(r.Header.Get("X-Forwarded-Proto")) == "https" {
		return true
	}
	return false
}
