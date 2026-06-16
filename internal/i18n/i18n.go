package i18n

import (
	"net/http"
	"strings"
)

// SupportedLangs lists the available UI languages.
var SupportedLangs = []string{"en", "it"}

// T returns the translation for key in lang, falling back to English.
func T(lang, key string) string {
	if m, ok := translations[lang]; ok {
		if v, ok := m[key]; ok {
			return v
		}
	}
	if v, ok := translations["en"][key]; ok {
		return v
	}
	return key
}

// Detect returns the best matching language from a request.
// Priority: ?lang= param > agentroom_lang cookie > Accept-Language header > default "en".
func Detect(r *http.Request, defaultLang string) string {
	// Query param (also used by the switcher link)
	if l := r.URL.Query().Get("lang"); isSupported(l) {
		return l
	}
	// Cookie
	if c, err := r.Cookie("agentroom_lang"); err == nil && isSupported(c.Value) {
		return c.Value
	}
	// Accept-Language header
	if al := r.Header.Get("Accept-Language"); al != "" {
		for _, tag := range strings.Split(al, ",") {
			tag = strings.TrimSpace(strings.SplitN(tag, ";", 2)[0])
			tag = strings.ToLower(strings.SplitN(tag, "-", 2)[0])
			if isSupported(tag) {
				return tag
			}
		}
	}
	if isSupported(defaultLang) {
		return defaultLang
	}
	return "en"
}

func isSupported(lang string) bool {
	for _, l := range SupportedLangs {
		if l == lang {
			return true
		}
	}
	return false
}
