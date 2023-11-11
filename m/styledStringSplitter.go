package m

import (
	"strings"
	"unicode/utf8"

	"github.com/walles/moar/twin"
)

const esc = '\x1b'

type styledStringSplitter struct {
	input             string
	nextByteIndex     int
	previousByteIndex int
	style             twin.Style

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
			return
		}

		if char == esc {
			escIndex := s.previousByteIndex
			success := s.handleEscape()
			if !success {
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
	if len(s.parts) == 0 {
		// We just got started
		s.parts = append(s.parts, _StyledString{
			String: "",
			Style:  twin.StyleDefault,
		})
	}

	lastPart := &s.parts[len(s.parts)-1]
	lastPart.String += string(char)
}

func (s *styledStringSplitter) handleEscape() bool {
	char := s.nextChar()
	if char == '[' || char == ']' {
		// Got the start of a CSI or an OSC sequence
		return s.consumeControlSequence(char)
	}

	return false
}

func (s *styledStringSplitter) consumeControlSequence(charAfterEsc rune) bool {
	// Points to right after "ESC["
	startIndex := s.nextByteIndex

	// We're looking for a letter to end the CSI sequence
	for {
		char := s.nextChar()
		if char == -1 {
			return false
		}

		if char == ';' || (char >= '0' && char <= '9') {
			// Sequence still in progress
			continue
		}

		// The end, handle what we got
		endIndexExclusive := s.nextByteIndex
		return s.handleCompleteControlSequence(charAfterEsc, s.input[startIndex:endIndexExclusive])
	}
}

// If the whole CSI sequence is ESC[33m, you should call this function with just
// "33m".
func (s *styledStringSplitter) handleCompleteControlSequence(charAfterEsc rune, sequence string) bool {
	if charAfterEsc == ']' {
		return s.handleCompleteOscSequence(sequence)
	}

	lastChar := sequence[len(sequence)-1]
	if lastChar == 'm' {
		newStyle := rawUpdateStyle(s.style, sequence)
		s.startNewPart(newStyle)
		return true
	}

	return false
}

func (s *styledStringSplitter) handleCompleteOscSequence(sequence string) bool {
	if sequence == "K" || sequence == "0K" {
		// Clear to end of line
		s.trailer = s.style
		return true
	}

	if sequence == "8;" {
		return s.handleUrl()
	}

	return false
}

// We just got ESC]8; and should now read the URL. URLs end with ASCII 7 BEL or ESC \.
func (s *styledStringSplitter) handleUrl() bool {
	// Valid URL characters.
	// Ref: https://stackoverflow.com/a/1547940/473672
	const validChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-._~:/?#[]@!$&'()*+,;="

	// Points to right after "ESC]8;"
	urlStartIndex := s.nextByteIndex

	justSawEsc := false
	for {
		char := s.nextChar()
		if char == -1 {
			return false
		}

		if justSawEsc {
			if char != '\\' {
				return false
			}

			// End of URL
			urlEndIndexExclusive := s.nextByteIndex - 2
			url := s.input[urlStartIndex:urlEndIndexExclusive]
			s.startNewPart(s.style.WithHyperlink(&url))
			return true
		}

		// Invariant: justSawEsc == false

		if char == esc {
			justSawEsc = true
			continue
		}

		if char == '\x07' {
			// End of URL
			urlEndIndexExclusive := s.nextByteIndex - 2
			url := s.input[urlStartIndex:urlEndIndexExclusive]
			s.startNewPart(s.style.WithHyperlink(&url))
			return true
		}

		if !strings.ContainsRune(validChars, char) {
			return false
		}

		// It's a valid URL char, keep going
	}
}

func (s *styledStringSplitter) startNewPart(style twin.Style) {
	if len(s.parts) > 0 && s.parts[len(s.parts)-1].Style == style {
		// Last part already matches the new style, never mind
		return
	}

	s.parts = append(s.parts, _StyledString{
		String: "",
		Style:  style,
	})
}
