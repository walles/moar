package twin

import (
	"testing"

	"gotest.tools/v3/assert"
)

func assertEncode(t *testing.T, incomingString string, expectedEvent Event, expectedRemainder string) {
	actualEvent, actualRemainder := consumeEncodedEvent(incomingString)
	assert.Equal(t, *actualEvent, expectedEvent)
	assert.Equal(t, actualRemainder, expectedRemainder)
}

func TestConsumeEncodedEvent(t *testing.T) {
	assertEncode(t, "a", EventRune{rune: 'a'}, "")
}
