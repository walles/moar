package reader

import (
	"strings"
	"testing"

	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/styles"
	"gotest.tools/v3/assert"
)

func TestPauseAfterNLines(t *testing.T) {
	pauseAfterLines := 1

	// Get ourselves a reader
	twoLines := strings.NewReader("one\ntwo\n")
	testMe, err := NewFromStream(
		"TestPauseAfterNLines",
		twoLines,
		formatters.TTY,
		ReaderOptions{
			PauseAfterLines: &pauseAfterLines,
			Style:           styles.Get("native"),
		})
	assert.NilError(t, err)

	// FIXME: Expect a pause notification

	// FIXME: Check that the reader has exactly the first line and nothing else

	// FIXME: Tell reader to continue

	// FIXME: Expect an unpause notification

	// FIXME: Expect a done notification

	// FIXME: Check that the reader has both lines and nothing else
}
