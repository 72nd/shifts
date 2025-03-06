package web

import (
	"strings"

	"github.com/microcosm-cc/bluemonday"
)

var sanitizer *bluemonday.Policy

func init() {
	// Initialisiere den Sanitizer einmal beim Start
	sanitizer = bluemonday.StrictPolicy()
}

// SanitizeText entfernt potenziell gefährliche HTML-Tags und Attribute aus dem Text
// und verwendet dafür die bluemonday-Bibliothek
func SanitizeText(text string) string {
	if text == "" {
		return ""
	}

	// Trimme Leerzeichen und sanitisiere den Text
	text = strings.TrimSpace(text)
	return sanitizer.Sanitize(text)
}
