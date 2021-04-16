// Package twin provides Terminal Window interaction
package twin

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	log "github.com/sirupsen/logrus"
	"golang.org/x/term"
)

type Screen interface {
	// Close() restores terminal to normal state, must be called after you are
	// done with your screen
	Close()

	Clear()

	SetCell(column int, row int, cell Cell)

	// Render our contents into the terminal window
	Show()

	// Returns screen width and height.
	//
	// NOTE: Never cache this response! On window resizes you'll get an
	// EventResize on the Screen.Events channel, and this method will start
	// returning the new size instead.
	Size() (int, int)

	// ShowCursorAt() moves the cursor to the given screen position and makes
	// sure it is visible.
	//
	// If the position is outside of the screen, the cursor will be hidden.
	ShowCursorAt(column int, row int)

	// This channel is what your main loop should be checking.
	Events() chan Event
}

type UnixScreen struct {
	widthAccessFromSizeOnly  int // Access from Size() method only
	heightAccessFromSizeOnly int // Access from Size() method only
	cells                    [][]Cell

	// Note that the type here doesn't matter, we only want to know whether or
	// not this channel has been signalled
	sigwinch chan int

	events           chan Event
	oldTerminalState *term.State

	ttyIn  *os.File
	ttyOut *os.File
}

// Cell is a rune with a style to be written to a cell on screen
type Cell struct {
	Rune  rune
	Style Style
}

func (cell Cell) String() string {
	return fmt.Sprint("rune='", string(cell.Rune), "' ", cell.Style)
}

// Example event: "\x1b[<65;127;41M"
//
// Where:
//
// * "\x1b[<" says this is a mouse event
//
// * "65" says this is Wheel Up. "64" would be Wheel Down.
//
// * "127" is the column number on screen, "1" is the first column.
//
// * "41" is the row number on screen, "1" is the first row.
//
// * "M" marks the end of the mouse event.
var MOUSE_EVENT_REGEX = regexp.MustCompile("^\x1b\\[<([0-9]+);([0-9]+);([0-9]+)M")

// NewScreen() requires Close() to be called after you are done with your new
// screen, most likely somewhere in your shutdown code.
func NewScreen() (Screen, error) {
	screen := UnixScreen{}

	// The number "80" here is from manual testing on my MacBook:
	//
	// First, start "./moar.sh sample-files/large-git-log-patch.txt".
	//
	// Then do a two finger flick initiating a momentum based scroll-up.
	//
	// Now, if you get "Events buffer full" warnings, the buffer is too small.
	//
	// By this definition, 40 was too small, and 80 was OK.
	screen.events = make(chan Event, 80)

	screen.setupSigwinchNotification()

	// os.Stdout is a stream that goes to our terminal window.
	//
	// So if we read from there, we'll get input from the terminal window.
	//
	// Reading from os.Stdin will fail if we're getting data piped into
	// ourselves from some other command.
	//
	// Tested on macOS and Linux, works like a charm!
	screen.ttyIn = os.Stdout // <- YES, WE SHOULD ASSIGN STDOUT TO TTYIN
	screen.ttyOut = os.Stdout

	// Set input stream to raw mode
	var err error
	screen.oldTerminalState, err = term.MakeRaw(int(screen.ttyIn.Fd()))
	if err != nil {
		panic(err)
	}

	screen.setAlternateScreenMode(true)
	screen.enableMouseTracking(true)
	screen.hideCursor(true)

	go screen.mainLoop()

	return &screen, nil
}

// Close() restores terminal to normal state, must be called after you are done
// with the screen returned by NewScreen()
func (screen *UnixScreen) Close() {
	screen.hideCursor(false)
	screen.enableMouseTracking(false)
	screen.setAlternateScreenMode(false)
	term.Restore(int(screen.ttyIn.Fd()), screen.oldTerminalState)
}

func (screen *UnixScreen) Events() chan Event {
	return screen.events
}

// Write string to ttyOut, panic on failure, return number of bytes written.
func (screen *UnixScreen) write(string string) int {
	bytesWritten, err := screen.ttyOut.Write([]byte(string))
	if err != nil {
		panic(err)
	}
	return bytesWritten
}

func (screen *UnixScreen) setAlternateScreenMode(enable bool) {
	// Ref: https://stackoverflow.com/a/11024208/473672
	if enable {
		screen.write("\x1b[?1049h")
	} else {
		screen.write("\x1b[?1049l")
	}
}

func (screen *UnixScreen) hideCursor(hide bool) {
	// Ref: https://en.wikipedia.org/wiki/ANSI_escape_code#CSI_(Control_Sequence_Introducer)_sequences
	if hide {
		screen.write("\x1b[?25l")
	} else {
		screen.write("\x1b[?25h")
	}
}

func (screen *UnixScreen) enableMouseTracking(enable bool) {
	if enable {
		screen.write("\x1b[?1006;1000h")
	} else {
		screen.write("\x1b[?1006;1000l")
	}
}

// ShowCursorAt() moves the cursor to the given screen position and makes sure
// it is visible.
//
// If the position is outside of the screen, the cursor will be hidden.
func (screen *UnixScreen) ShowCursorAt(column int, row int) {
	if column < 0 {
		screen.hideCursor(true)
		return
	}
	if row < 0 {
		screen.hideCursor(true)
		return
	}

	width, height := screen.Size()
	if column >= width {
		screen.hideCursor(true)
		return
	}
	if row >= height {
		screen.hideCursor(true)
		return
	}

	// https://en.wikipedia.org/wiki/ANSI_escape_code#CSI_(Control_Sequence_Introducer)_sequences
	screen.write(fmt.Sprintf("\x1b[%d;%dH", row, column))
	screen.hideCursor(false)
}

func (screen *UnixScreen) mainLoop() {
	// "1400" comes from me trying fling scroll operations on my MacBook
	// trackpad and looking at the high watermark (logged below).
	//
	// The highest I saw when I tried this was 700 something. 1400 is twice
	// that, so 1400 should be good.
	buffer := make([]byte, 1400)

	maxBytesRead := 0
	for {
		count, err := screen.ttyIn.Read(buffer)
		if err != nil {
			panic(err)
		}

		if count > maxBytesRead {
			maxBytesRead = count
			log.Trace("ttyin high watermark bumped to ", maxBytesRead, " bytes")
		}

		encodedKeyCodeSequences := string(buffer[0:count])
		if !utf8.ValidString(encodedKeyCodeSequences) {
			log.Warn("Got invalind UTF-8 sequence on ttyin: ", encodedKeyCodeSequences)
			continue
		}

		for len(encodedKeyCodeSequences) > 0 {
			var event *Event
			event, encodedKeyCodeSequences = consumeEncodedEvent(encodedKeyCodeSequences)

			if event == nil {
				// No event, go wait for more
				break
			}

			// Post the event
			select {
			case screen.events <- *event:
				// Yay
			default:
				// If this happens, consider increasing the channel size in
				// NewScreen()
				log.Warn("Events buffer full, events are being dropped")
			}
		}
	}
}

// Consume initial key code from the sequence of encoded keycodes.
//
// Returns a (possibly nil) event that should be posted, and the remainder of
// the encoded events sequence.
func consumeEncodedEvent(encodedEventSequences string) (*Event, string) {
	for singleKeyCodeSequence, keyCode := range escapeSequenceToKeyCode {
		if !strings.HasPrefix(encodedEventSequences, singleKeyCodeSequence) {
			continue
		}

		// Encoded key code sequence found, report it!
		var event Event = EventKeyCode{keyCode}
		return &event, strings.TrimPrefix(encodedEventSequences, singleKeyCodeSequence)
	}

	mouseMatch := MOUSE_EVENT_REGEX.FindStringSubmatch(encodedEventSequences)
	if mouseMatch != nil {
		if mouseMatch[1] == "65" {
			var event Event = EventMouse{buttons: MouseWheelDown}
			return &event, strings.TrimPrefix(encodedEventSequences, mouseMatch[0])
		}
		if mouseMatch[1] == "64" {
			var event Event = EventMouse{buttons: MouseWheelUp}
			return &event, strings.TrimPrefix(encodedEventSequences, mouseMatch[0])
		}

		log.Debug("Unhandled multi character mouse escape sequence(s): ", encodedEventSequences)
		return nil, ""
	}

	// No escape sequence prefix matched
	runes := []rune(encodedEventSequences)
	if len(runes) != 1 {
		// This means one or more sequences should be added to
		// escapeSequenceToKeyCode in keys.go.
		log.Debug("Unhandled multi character terminal escape sequence(s): ", encodedEventSequences)

		// Mark everything as consumed since we don't know how to proceed otherwise.
		return nil, ""
	}

	var event Event
	if runes[0] == '\x1b' {
		event = EventKeyCode{KeyEscape}
	} else if runes[0] == '\r' {
		event = EventKeyCode{KeyEnter}
	} else {
		// Report the single rune
		event = EventRune{rune: runes[0]}
	}

	return &event, ""
}

// Returns screen width and height.
//
// NOTE: Never cache this response! On window resizes you'll get an EventResize
// on the Screen.Events channel, and this method will start returning the new
// size instead.
func (screen *UnixScreen) Size() (int, int) {
	select {
	case <-screen.sigwinch:
		// Resize logic needed, see below
	default:
		// No resize, go with the existing values
		return screen.widthAccessFromSizeOnly, screen.heightAccessFromSizeOnly
	}

	// Window was resized
	width, height, err := term.GetSize(int(screen.ttyOut.Fd()))
	if err != nil {
		panic(err)
	}

	if screen.widthAccessFromSizeOnly == width && screen.heightAccessFromSizeOnly == height {
		// Not sure when this would happen, but if it does this wasn't really a
		// resize, and we don't need to treat it as such.
		return screen.widthAccessFromSizeOnly, screen.heightAccessFromSizeOnly
	}

	newCells := make([][]Cell, height)
	for rowNumber := 0; rowNumber < height; rowNumber++ {
		newCells[rowNumber] = make([]Cell, width)
	}

	// FIXME: Copy any existing contents over to the new, resized screen array
	// FIXME: Fill any non-initialized cells with whitespace

	screen.widthAccessFromSizeOnly = width
	screen.heightAccessFromSizeOnly = height
	screen.cells = newCells

	return screen.widthAccessFromSizeOnly, screen.heightAccessFromSizeOnly
}

func (screen *UnixScreen) SetCell(column int, row int, cell Cell) {
	if column < 0 {
		return
	}
	if row < 0 {
		return
	}

	width, height := screen.Size()
	if column >= width {
		return
	}
	if row >= height {
		return
	}
	screen.cells[row][column] = cell
}

func (screen *UnixScreen) Clear() {
	empty := NewCell(' ', StyleDefault)

	width, height := screen.Size()
	for row := 0; row < height; row++ {
		for column := 0; column < width; column++ {
			screen.cells[row][column] = empty
		}
	}
}

// Returns the rendered line, plus how many information carrying cells went into
// it (see headerLength inside of this function).
func renderLine(row []Cell) (string, int) {
	width := len(row)
	if width == 0 {
		return "", 0
	}

	lastCell := row[len(row)-1]

	// How many trailing whitespace characters are there matching lastCell?
	trailerLength := 0
	if lastCell.Style.attrs.has(AttrBlink) {
		// This block intentionally left blank.
		//
		// Trailer is rendered by clearing to EOL, and AttrBlink will most
		// likely not survive that. Don't bother with any trailer in this case.
	} else if lastCell.Style.attrs.has(AttrReverse) {
		// This block intentionally left blank.
		//
		// Trailer is rendered by clearing to EOL, and AttrReverse didn't
		// survive that when I tried it using iTerm2 3.4.4 on my MacBook. Don't
		// bother with any trailer in this case.
	} else if lastCell.Rune == ' ' {
		// Line has a number of trailing spaces
		for i := len(row) - 1; i >= 0; i-- {
			currentCell := row[i]
			if currentCell != lastCell {
				break
			}
			trailerLength++
		}
	}

	// How many information carrying cells are there before the trailer?
	headerLength := len(row) - trailerLength

	var builder strings.Builder

	// Set initial line style to normal
	builder.WriteString("\x1b[m")
	lastStyle := StyleDefault

	for column := 0; column < headerLength; column++ {
		cell := row[column]

		style := cell.Style
		runeToWrite := cell.Rune
		if !unicode.IsPrint(runeToWrite) {
			// Highlight unprintable runes
			style = Style{
				fg:    NewColor16(7), // White
				bg:    NewColor16(1), // Red
				attrs: AttrBold,
			}
			runeToWrite = '?'
		}

		if style != lastStyle {
			builder.WriteString(style.RenderUpdateFrom(lastStyle))
			lastStyle = style
		}

		builder.WriteRune(runeToWrite)
	}

	// Set trailer attributes
	builder.WriteString(lastCell.Style.RenderUpdateFrom(lastStyle))

	// Clear to end of line
	// https://en.wikipedia.org/wiki/ANSI_escape_code#CSI_(Control_Sequence_Introducer)_sequences
	builder.WriteString("\x1b[K")

	if lastStyle != StyleDefault {
		// Reset style after each line
		builder.WriteString("\x1b[m")
	}

	return builder.String(), headerLength
}

// Render our contents into the terminal window
//
// Note that we start by prepping everything we want to write, then write it all
// in one go. This is to make the screen update experience as atomic and flicker
// free as possible.
func (screen *UnixScreen) Show() {
	var builder strings.Builder

	// Start in the top left corner:
	// https://en.wikipedia.org/wiki/ANSI_escape_code#CSI_(Control_Sequence_Introducer)_sequences
	builder.WriteString("\x1b[1;1H")

	_, height := screen.Size()
	for row := 0; row < height; row++ {
		rendered, lineLength := renderLine(screen.cells[row])
		builder.WriteString(rendered)

		lastLine := row == (height - 1)
		if lineLength < len(screen.cells[row]) && !lastLine {
			builder.WriteString("\r\n")
		}
	}

	// Write out what we have
	t0 := time.Now()
	byteCount := screen.write(builder.String())
	t1 := time.Now()
	dt := t1.Sub(t0)
	megaBytesPerSecond := float64(byteCount) / (1024.0 * 1024.0) / dt.Seconds()
	log.Trace("Wrote ", byteCount, " bytes to screen in ", dt, " at ", int(megaBytesPerSecond), "MB/s")
}

func NewCell(rune rune, style Style) Cell {
	return Cell{
		Rune:  rune,
		Style: style,
	}
}
