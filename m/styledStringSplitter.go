package m

import (
	"fmt"
	"strings"
	"unicode/utf8"

	log "github.com/sirupsen/logrus"
	"github.com/walles/moar/twin"
)

const esc = '\x1b'

type styledStringSplitter struct {
	input             string
	nextByteIndex     int
	previousByteIndex int

	inProgressString strings.Builder
	inProgressStyle  twin.Style

	parts   []_StyledString
	trailer twin.Style
}

func styledStringsFromString(s string) styledStringsWithTrailer {
	if !strings.ContainsAny(s, "\x1b") {
		// This shortcut makes BenchmarkPlainTextSearch() perform a lot better
		return styledStringsWithTrailer{
			trailer: twin.StyleDefault,
			styledStrings: []_StyledString{{
				String: s,
				Style:  twin.StyleDefault,
			}},
		}
	}

	splitter := styledStringSplitter{
		input: s,
	}
	splitter.run()

	return styledStringsWithTrailer{
		trailer:       splitter.trailer,
		styledStrings: splitter.parts,
	}
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
				failed := s.input[escIndex:s.nextByteIndex]
				log.Debug("Failed to parse <", strings.ReplaceAll(failed, "\x1b", "ESC"), ">: ", err)

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

	return fmt.Errorf("Unhandled char after ESC: %q", char)
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

		if char == ';' || char == ':' || (char >= '0' && char <= '9') {
			// Sequence still in progress

			if charAfterEsc == ']' && s.input[startIndex:s.nextByteIndex] == "8;;" {
				// Special case, here comes the URL
				return s.handleUrl()
			}

			continue
		}

		// The end, handle what we got
		endIndexExclusive := s.nextByteIndex
		return s.handleCompleteControlSequence(charAfterEsc, s.input[startIndex:endIndexExclusive])
	}
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
		newStyle, err := rawUpdateStyle(s.inProgressStyle, sequence)
		if err != nil {
			return err
		}

		s.startNewPart(newStyle)
		return nil
	}

	return fmt.Errorf("Expected 'm' at the end of the control sequence, got %q", lastChar)
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
			} else {
				return fmt.Errorf("Expected ESC \\ after ESC]133;X, got %q", s.lastChar())
			}
		}
	}

	return fmt.Errorf("Unhandled OSC sequence")
}

// We just got ESC]8; and should now read the URL. URLs end with ASCII 7 BEL or ESC \.
func (s *styledStringSplitter) handleUrl() error {
	// Valid URL characters.
	// Ref: https://stackoverflow.com/a/1547940/473672
	const validChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-._~:/?#[]@!$&'()*+,;="

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

		if !strings.ContainsRune(validChars, char) {
			return fmt.Errorf("Invalid URL character: %q", char)
		}

		// It's a valid URL char, keep going
	}
}

func (s *styledStringSplitter) startNewPart(style twin.Style) {
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

	s.parts = append(s.parts, _StyledString{
		String: s.inProgressString.String(),
		Style:  s.inProgressStyle,
	})
}
