package twin

import (
	"strings"
	"testing"

	"gotest.tools/v3/assert"
)

func assertEncode(t *testing.T, incomingString string, expectedEvent Event, expectedRemainder string) {
	message := strings.Replace(incomingString, "\x1b", "ESC", -1)
	message = strings.Replace(message, "\r", "RET", -1)

	actualEvent, actualRemainder := consumeEncodedEvent(incomingString)
	assert.Equal(t, *actualEvent, expectedEvent,
		"Input: %s Result: %#v Expected: %#v", message, *actualEvent, expectedEvent)
	assert.Equal(t, actualRemainder, expectedRemainder, message)
}

func TestConsumeEncodedEvent(t *testing.T) {
	assertEncode(t, "a", EventRune{rune: 'a'}, "")
	assertEncode(t, "\r", EventKeyCode{keyCode: KeyEnter}, "")
	assertEncode(t, "\x1b", EventKeyCode{keyCode: KeyEscape}, "")

	// Implicitly test having a remaining rune at the end
	assertEncode(t, "\x1b[Ax", EventKeyCode{keyCode: KeyUp}, "x")

	assertEncode(t, "\x1b[<64;127;41M", EventMouse{buttons: MouseWheelUp}, "")
	assertEncode(t, "\x1b[<65;127;41M", EventMouse{buttons: MouseWheelDown}, "")
}
