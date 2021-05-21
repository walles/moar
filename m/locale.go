package m

import (
	"os"
	"strings"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// Like "fmt" but for the current locale
var lfmt = getLocaleFormatter()

func parseLocaleString(localeString string) *language.Tag {
	dotIndex := strings.IndexRune(localeString, '.')
	if dotIndex >= 0 {
		// Turns "sv_SE.UTF-8" into "sv_SE"
		localeString = localeString[:dotIndex]
	}

	candidate, err := language.Parse(localeString)
	if err != nil {
		return nil
	}

	if candidate == language.Und {
		return nil
	}

	return &candidate
}

func getLocaleFormatter() *message.Printer {
	// Prefer LC_NUMERIC value, otherwise LANG value
	locale := language.Und

	candidate := parseLocaleString(os.Getenv("LANG"))
	if candidate != nil {
		// Overwrite locale selection with LANG parse result
		locale = *candidate
	}

	candidate = parseLocaleString(os.Getenv("LC_NUMERIC"))
	if candidate != nil {
		// Overwrite locale selection with LC_NUMERIC parse result
		locale = *candidate
	}

	return message.NewPrinter(locale)
}
