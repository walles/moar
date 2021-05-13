package m

import (
	"bytes"
	"os"
	"strings"

	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/lexers"

	log "github.com/sirupsen/logrus"
)

// Files larger than this won't be highlighted
const MAX_HIGHLIGHT_SIZE int64 = 1024 * 1024

// Read and highlight a file using Chroma: https://github.com/alecthomas/chroma
//
// If force is true, file will always be highlighted. If force is false, files
// larger than MAX_HIGHLIGHT_SIZE will not be highlighted.
//
// Returns nil with no error if highlighting would be a no-op.
func highlight(filename string, force bool, style chroma.Style, formatter chroma.Formatter) (*string, error) {
	// Highlight input file using Chroma:
	// https://github.com/alecthomas/chroma
	fileInfo, err := os.Stat(filename)
	if err != nil {
		return nil, err
	}
	if fileInfo.Size() > MAX_HIGHLIGHT_SIZE {
		log.Debugf("Not highlighting %s because it is %d bytes large, which is larger than moar's built-in highlighting limit of %d bytes",
			filename, fileInfo.Size(), MAX_HIGHLIGHT_SIZE)
		return nil, nil
	}

	lexer := lexers.Match(filename)
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

	contents, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	iterator, err := lexer.Tokenise(nil, string(contents))
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
