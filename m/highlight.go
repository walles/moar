package m

import (
	"bytes"
	"strings"

	"github.com/alecthomas/chroma/v2"
)

// Read and highlight some text using Chroma:
// https://github.com/alecthomas/chroma
//
// Returns nil with no error if highlighting would be a no-op.
func highlight(text string, style chroma.Style, formatter chroma.Formatter, lexer chroma.Lexer) (*string, error) {
	if lexer == nil {
		// No highlighter available for this file type
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

	iterator, err := lexer.Tokenise(nil, text)
	if err != nil {
		return nil, err
	}

	var stringBuffer bytes.Buffer
	err = formatter.Format(&stringBuffer, &style, iterator)
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
