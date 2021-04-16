package twin

type KeyCode uint16

const (
	KeyEscape KeyCode = iota
	KeyEnter

	KeyBackspace
	KeyDelete

	KeyUp
	KeyDown
	KeyRight
	KeyLeft

	KeyHome
	KeyEnd
	KeyPgUp
	KeyPgDown
)

// Map incoming escape keystrokes to keycodes, used in consumeEncodedEvent() in
// screen.go.
//
// NOTE: If you put a single ESC character in here ('\x1b') it will be consumed
// by itself rather than as part of the sequence it belongs to, and parsing of
// all special sequences starting with ESC will break down.
//
// FIXME: Write a test preventing that from happening.
var escapeSequenceToKeyCode = map[string]KeyCode{
	// NOTE: Please keep this list in the same order as the KeyCode const()
	// section above.

	// KeyEscape intentionally left out because it's too short, see comment
	// above.

	// KeyEnter intentionally left out because it's too short, see comment
	// above.

	"\x7f":    KeyBackspace,
	"\x1b[3~": KeyDelete,

	"\x1b[A": KeyUp,
	"\x1b[B": KeyDown,
	"\x1b[C": KeyRight,
	"\x1b[D": KeyLeft,

	"\x1b[H":  KeyHome,
	"\x1b[F":  KeyEnd,
	"\x1b[5~": KeyPgUp,
	"\x1b[6~": KeyPgDown,
}
