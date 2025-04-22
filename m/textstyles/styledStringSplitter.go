package textstyles

import (
	"fmt"
	"strings"
	"unicode/utf8"

	log "github.com/sirupsen/logrus"
	"github.com/walles/moar/m/linenumbers"
	"github.com/walles/moar/twin"
)

const esc = '\x1b'

type styledStringSplitter struct {
	input          string
	lineNumber     *linenumbers.LineNumber
	plainTextStyle twin.Style

	nextByteIndex     int
	previousByteIndex int

	inProgressString strings.Builder
	inProgressStyle  twin.Style
	numbersBuffer    []uint

	trailer twin.Style

	callback func(str string, style twin.Style)
}

// Returns the style of the line's trailer
func styledStringsFromString(plainTextStyle twin.Style, s string, lineNumber *linenumbers.LineNumber, callback func(string, twin.Style)) twin.Style {
	if !strings.ContainsAny(s, "\x1b") {
		// This shortcut makes BenchmarkPlainTextSearch() perform a lot better
		callback(s, plainTextStyle)
		return plainTextStyle
	}

	splitter := styledStringSplitter{
		input:           s,
		lineNumber:      lineNumber,
		plainTextStyle:  plainTextStyle, // How plain text should be styled
		inProgressStyle: plainTextStyle, // Plain text style until something else comes along
		callback:        callback,
		trailer:         plainTextStyle, // Plain text style until something else comes along
	}
	splitter.run()

	return splitter.trailer
}

func (s *styledStringSplitter) nextChar() rune {
	if s.nextByteIndex >= len(s.input) {
		s.previousByteIndex = s.nextByteIndex
		return -1
	}

	char, size := utf8.DecodeRuneInString(s.input[s.nextByteIndex:])
	s.previousByteIndex = s.nextByteIndex
	s.nextByteIndex += size
	return char
}

// Returns whatever the last call to nextChar() returned
func (s *styledStringSplitter) lastChar() rune {
	if s.previousByteIndex >= len(s.input) {
		return -1
	}

	char, _ := utf8.DecodeRuneInString(s.input[s.previousByteIndex:])
	return char
}

func (s *styledStringSplitter) run() {
	char := s.nextChar()
	for {
		if char == -1 {
			s.finalizeCurrentPart()
			return
		}

		if char == esc {
			escIndex := s.previousByteIndex
			err := s.handleEscape()
			if err != nil {
				header := ""
				if s.lineNumber != nil {
					header = fmt.Sprintf("Line %s: ", s.lineNumber.Format())
				}

				failed := s.input[escIndex:s.nextByteIndex]
				log.Debug(header, "<", strings.ReplaceAll(failed, "\x1b", "ESC"), ">: ", err)

				// Somewhere in handleEscape(), we got a character that was
				// unexpected. We need to treat everything up to before that
				// character as just plain runes.
				for _, char := range s.input[escIndex:s.previousByteIndex] {
					s.handleRune(char)
				}

				// Start over with the character that caused the problem
				char = s.lastChar()
				continue
			}
		} else {
			s.handleRune(char)
		}

		char = s.nextChar()
	}
}

func (s *styledStringSplitter) handleRune(char rune) {
	s.inProgressString.WriteRune(char)
}

func (s *styledStringSplitter) handleEscape() error {
	char := s.nextChar()
	if char == '[' || char == ']' {
		// Got the start of a CSI or an OSC sequence
		return s.consumeControlSequence(char)
	}
	if char == '(' {
		// Designate G0 charset: https://www.xfree86.org/4.8.0/ctlseqs.html
		return s.consumeG0Charset()
	}

	return fmt.Errorf("Unhandled Fe sequence ESC%c", char)
}

func (s *styledStringSplitter) consumeControlSequence(charAfterEsc rune) error {
	// Points to right after "ESC["
	startIndex := s.nextByteIndex

	// We're looking for a letter to end the CSI sequence
	for {
		char := s.nextChar()
		if char == -1 {
			return fmt.Errorf("Line ended in the middle of a control sequence")
		}

		// Range from here:
		// https://en.wikipedia.org/wiki/ANSI_escape_code#CSI_(Control_Sequence_Introducer)_sequences
		if char >= 0x30 && char <= 0x3f {
			// Sequence still in progress

			if charAfterEsc == ']' && s.input[startIndex:s.nextByteIndex] == "8;;" {
				// Special case, here comes the URL
				return s.handleURL()
			}

			continue
		}

		// The end, handle what we got
		endIndexExclusive := s.nextByteIndex
		return s.handleCompleteControlSequence(charAfterEsc, s.input[startIndex:endIndexExclusive])
	}
}

func (s *styledStringSplitter) consumeG0Charset() error {
	// First char after "ESC("
	char := s.nextChar()
	if char == 'B' {
		// G0 charset is now "B" (ASCII)
		s.startNewPart(s.plainTextStyle)
		return nil
	}

	return fmt.Errorf("Unhandled G0 charset: %c", char)
}

// If the whole CSI sequence is ESC[33m, you should call this function with just
// "33m".
func (s *styledStringSplitter) handleCompleteControlSequence(charAfterEsc rune, sequence string) error {
	if charAfterEsc == ']' {
		return s.handleOsc(sequence)
	}

	if charAfterEsc != '[' {
		return fmt.Errorf("Unexpected charAfterEsc: %c", charAfterEsc)
	}

	if sequence == "K" || sequence == "0K" {
		// Clear to end of line
		s.trailer = s.inProgressStyle
		return nil
	}

	lastChar := sequence[len(sequence)-1]
	if lastChar == 'm' {
		var newStyle twin.Style
		var err error
		newStyle, s.numbersBuffer, err = rawUpdateStyle(s.inProgressStyle, sequence, s.numbersBuffer)
		if err != nil {
			return err
		}

		s.startNewPart(newStyle)
		return nil
	}

	return fmt.Errorf("Unhandled CSI type %q", lastChar)
}

func (s *styledStringSplitter) handleOsc(sequence string) error {
	if strings.HasPrefix(sequence, "133;") && len(sequence) == len("133;A") {
		// Got ESC]133;X, where "X" could be anything. These are prompt hints,
		// and rendering those makes no sense. We should just ignore them:
		// https://gitlab.freedesktop.org/Per_Bothner/specifications/blob/master/proposals/semantic-prompts.md
		endMarker := s.nextChar()
		if endMarker == '\x07' {
			return nil
		}

		if endMarker == esc {
			if s.nextChar() == '\\' {
				return nil
			}

			return fmt.Errorf("Expected ESC \\ after ESC]133;X, got %q", s.lastChar())
		}
	}

	return fmt.Errorf("Unhandled OSC sequence")
}

// Based on https://infra.spec.whatwg.org/#surrogate
func isSurrogate(char rune) bool {
	if char >= 0xD800 && char <= 0xDFFF {
		return true
	}

	return false
}

// Non-characters end with 0xfffe or 0xffff, or are in the range 0xFDD0 to 0xFDEF.
//
// Based on https://infra.spec.whatwg.org/#noncharacter
func isNonCharacter(char rune) bool {
	if char >= 0xFDD0 && char <= 0xFDEF {
		return true
	}

	// Non-characters end with 0xfffe or 0xffff
	if char&0xFFFF == 0xFFFE || char&0xFFFF == 0xFFFF {
		return true
	}

	return false
}

// Based on https://url.spec.whatwg.org/#url-code-points
func isValidURLChar(char rune) bool {
	if char == '\\' {
		// Ref: https://github.com/walles/moar/issues/244#issuecomment-2350908401
		return true
	}

	if char == '%' {
		// Needed for % escapes
		return true
	}

	// ASCII alphanumerics
	if char >= '0' && char <= '9' {
		return true
	}
	if char >= 'A' && char <= 'Z' {
		return true
	}
	if char >= 'a' && char <= 'z' {
		return true
	}

	if strings.ContainsRune("!$&'()*+,-./:;=?@_~", char) {
		return true
	}

	if char < 0x00a0 {
		return false
	}

	if char > 0x10FFFD {
		return false
	}

	if isSurrogate(char) {
		return false
	}

	if isNonCharacter(char) {
		return false
	}

	return true
}

// We just got ESC]8; and should now read the URL. URLs end with ASCII 7 BEL or ESC \.
func (s *styledStringSplitter) handleURL() error {
	// Points to right after "ESC]8;"
	urlStartIndex := s.nextByteIndex

	justSawEsc := false
	for {
		char := s.nextChar()
		if char == -1 {
			return fmt.Errorf("Line ended in the middle of a URL")
		}

		if justSawEsc {
			if char != '\\' {
				return fmt.Errorf("Expected ESC \\ but got ESC %q", char)
			}

			// End of URL
			urlEndIndexExclusive := s.nextByteIndex - 2
			url := s.input[urlStartIndex:urlEndIndexExclusive]
			s.startNewPart(s.inProgressStyle.WithHyperlink(&url))
			return nil
		}

		// Invariant: justSawEsc == false

		if char == esc {
			justSawEsc = true
			continue
		}

		if char == '\x07' {
			// End of URL
			urlEndIndexExclusive := s.nextByteIndex - 1
			url := s.input[urlStartIndex:urlEndIndexExclusive]
			s.startNewPart(s.inProgressStyle.WithHyperlink(&url))
			return nil
		}

		if !isValidURLChar(char) {
			return fmt.Errorf("Invalid URL character: <%q>", char)
		}

		// It's a valid URL char, keep going
	}
}

func (s *styledStringSplitter) startNewPart(style twin.Style) {
	if style == twin.StyleDefault {
		style = s.plainTextStyle
	}

	if style == s.inProgressStyle {
		// No need to start a new part
		return
	}

	s.finalizeCurrentPart()
	s.inProgressString.Reset()
	s.inProgressStyle = style
}

func (s *styledStringSplitter) finalizeCurrentPart() {
	if s.inProgressString.Len() == 0 {
		// Nothing to do
		return
	}

	s.callback(s.inProgressString.String(), s.inProgressStyle)
}
