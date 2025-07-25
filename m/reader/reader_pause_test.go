package reader

import (
	"strings"
	"testing"

	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/walles/moar/m/linemetadata"
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

	// Expect a pause notification since we configure it to pause after 1 line ^
	<-testMe.PauseStatusUpdated
	assert.Assert(t, testMe.PauseStatus.Load() == true,
		"Reader should be paused after reading %d lines", pauseAfterLines)

	// Verify that we have *not* received a done notification yet
	assert.Assert(t, testMe.Done.Load() == false,
		"Reader should not be done yet, only paused")

	// Check that the reader has exactly the first line and nothing else
	lines := testMe.GetLines(linemetadata.Index{}, 2).Lines
	assert.Equal(t, len(lines), 1,
		"Reader should have exactly one line after pausing")
	assert.Equal(t, lines[0].Plain(), "one",
		"Reader should have the first line after pausing")

	// Tell reader to continue
	testMe.SetPaused(false)

	// Expect an unpause notification
	<-testMe.PauseStatusUpdated
	assert.Assert(t, testMe.PauseStatus.Load() == false,
		"Reader should be unpaused after continuing")

	// Expect a done notification
	<-testMe.MaybeDone
	assert.Assert(t, testMe.Done.Load() == true,
		"Reader should be done after reading all lines")

	// Check that the reader has both lines
	lines = testMe.GetLines(linemetadata.Index{}, 3).Lines
	assert.Equal(t, len(lines), 2,
		"Reader should have two lines after unpausing")
	assert.Equal(t, lines[0].Plain(), "one",
		"Reader should have the first line after unpausing")
	assert.Equal(t, lines[1].Plain(), "two",
		"Reader should have the second line after unpausing")
}
