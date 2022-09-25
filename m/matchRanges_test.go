package m

import (
	"regexp"
	"testing"

	"gotest.tools/v3/assert"
)

// This is really a constant, don't change it!
var _TestString = "mamma"

func TestGetMatchRanges(t *testing.T) {
	matchRanges := getMatchRanges(&_TestString, regexp.MustCompile("m+"))
	assert.Equal(t, len(matchRanges.Matches), 2) // Two matches

	assert.DeepEqual(t, matchRanges.Matches[0][0], 0) // First match starts at 0
	assert.DeepEqual(t, matchRanges.Matches[0][1], 1) // And ends on 1 exclusive

	assert.DeepEqual(t, matchRanges.Matches[1][0], 2) // Second match starts at 2
	assert.DeepEqual(t, matchRanges.Matches[1][1], 4) // And ends on 4 exclusive
}

func TestGetMatchRangesNilPattern(t *testing.T) {
	matchRanges := getMatchRanges(&_TestString, nil)
	assert.Assert(t, matchRanges == nil)
	assert.Assert(t, !matchRanges.InRange(0))
}

func TestInRange(t *testing.T) {
	// Should match the one in TestGetMatchRanges()
	matchRanges := getMatchRanges(&_TestString, regexp.MustCompile("m+"))

	assert.Assert(t, !matchRanges.InRange(-1)) // Before start
	assert.Assert(t, matchRanges.InRange(0))   // m
	assert.Assert(t, !matchRanges.InRange(1))  // a
	assert.Assert(t, matchRanges.InRange(2))   // m
	assert.Assert(t, matchRanges.InRange(3))   // m
	assert.Assert(t, !matchRanges.InRange(4))  // a
	assert.Assert(t, !matchRanges.InRange(5))  // After end
}

func TestUtf8(t *testing.T) {
	// This test verifies that the match ranges are by rune rather than by byte
	unicodes := "-ä-ä-"
	matchRanges := getMatchRanges(&unicodes, regexp.MustCompile("ä"))

	assert.Assert(t, !matchRanges.InRange(0)) // -
	assert.Assert(t, matchRanges.InRange(1))  // ä
	assert.Assert(t, !matchRanges.InRange(2)) // -
	assert.Assert(t, matchRanges.InRange(3))  // ä
	assert.Assert(t, !matchRanges.InRange(4)) // -
}

func TestNoMatch(t *testing.T) {
	// This test verifies that the match ranges are by rune rather than by byte
	unicodes := "gris"
	matchRanges := getMatchRanges(&unicodes, regexp.MustCompile("apa"))

	assert.Assert(t, !matchRanges.InRange(0))
	assert.Assert(t, !matchRanges.InRange(1))
	assert.Assert(t, !matchRanges.InRange(2))
	assert.Assert(t, !matchRanges.InRange(3))
	assert.Assert(t, !matchRanges.InRange(4))
}

func TestEndMatch(t *testing.T) {
	// This test verifies that the match ranges are by rune rather than by byte
	unicodes := "-ä"
	matchRanges := getMatchRanges(&unicodes, regexp.MustCompile("ä"))

	assert.Assert(t, !matchRanges.InRange(0)) // -
	assert.Assert(t, matchRanges.InRange(1))  // ä
	assert.Assert(t, !matchRanges.InRange(2)) // Past the end
}

func TestRealWorldBug(t *testing.T) {
	// Verify a real world bug found in v1.9.8

	testString := "anna"
	matchRanges := getMatchRanges(&testString, regexp.MustCompile("n"))
	assert.Equal(t, len(matchRanges.Matches), 2) // Two matches

	assert.DeepEqual(t, matchRanges.Matches[0][0], 1) // First match starts at 1
	assert.DeepEqual(t, matchRanges.Matches[0][1], 2) // And ends on 2 exclusive

	assert.DeepEqual(t, matchRanges.Matches[1][0], 2) // Second match starts at 2
	assert.DeepEqual(t, matchRanges.Matches[1][1], 3) // And ends on 3 exclusive
}
