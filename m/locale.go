package m

import (
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// Like "fmt" but for the current locale
var lfmt = getLocaleFormatter()

func getLocaleFormatter() *message.Printer {
	// FIXME: Check LANG and LC_NUMERIC and parse it: https://pkg.go.dev/golang.org/x/text/message
	locale := language.English
	return message.NewPrinter(locale)
}
