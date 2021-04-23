package m

import (
	"bytes"
	"io/ioutil"
	"strings"

	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/formatters"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
)

// Read and highlight a file using Chroma: https://github.com/alecthomas/chroma
//
// Returns nil if highlighting would be a no-op.
func highlight(filename string) (*string, error) {
	// Highlight input file using Chroma:
	// https://github.com/alecthomas/chroma
	lexer := lexers.Match(filename)
	if lexer == nil {
		return nil, nil
	}

	// FIXME: Can we test for the lexer implementation class instead? That
	// should be more resilient towards this arbitrary string changing if we
	// upgrade Chroma at some point.
	if lexer.Config().Name == "plaintext" {
		// This highlighter doesn't provide any highlighting, but not doing
		// anything at all is cheaper and simpler, so we do that.
		return nil, nil
	}

	// See: https://github.com/alecthomas/chroma#identifying-the-language
	// FIXME: Do we actually need this? We should profile our reader performance
	// with and without.
	lexer = chroma.Coalesce(lexer)

	formatter := formatters.Get("terminal16m")
	if formatter == nil {
		formatter = formatters.Fallback
	}

	contents, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	iterator, err := lexer.Tokenise(nil, string(contents))
	if err != nil {
		return nil, err
	}

	var stringBuffer bytes.Buffer
	err = formatter.Format(&stringBuffer, styles.Native, iterator)
	if err != nil {
		return nil, err
	}

	highlighted := stringBuffer.String()

	// If buffer ends with SGR Reset ("<ESC>[0m"), remove it. Chroma sometimes
	// (always?) puts one of those by itself on the last line, making us believe
	// there is one line too many.
	sgrReset := "\x1b[0m"
	trimmed := strings.TrimSuffix(highlighted, sgrReset)

	return &trimmed, nil
}
