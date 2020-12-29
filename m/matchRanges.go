package m

import "regexp"

// MatchRanges collects match indices
type MatchRanges struct {
	Matches [][2]int
}

// GetMatchRanges locates one or more regexp matches in a string
func GetMatchRanges(String *string, Pattern *regexp.Regexp) *MatchRanges {
	if Pattern == nil {
		return nil
	}

	return &MatchRanges{
		Matches: toRunePositions(Pattern.FindAllStringIndex(*String, -1), String),
	}
}

// Convert byte indices to rune indices
func toRunePositions(byteIndices [][]int, matchedString *string) [][2]int {
	// FIXME: Will this function need to handle overlapping ranges?

	var returnMe [][2]int

	if len(byteIndices) == 0 {
		// Nothing to see here, move along
		return returnMe
	}

	fromByte := byteIndices[len(returnMe)][0]
	toByte := byteIndices[len(returnMe)][1]
	fromRune := -1
	runePosition := 0
	for bytePosition := range *matchedString {
		if fromByte == bytePosition {
			fromRune = runePosition
		}
		if toByte == bytePosition {
			toRune := runePosition
			returnMe = append(returnMe, [2]int{fromRune, toRune})

			fromRune = -1

			if len(returnMe) >= len(byteIndices) {
				// No more byte indices
				break
			}

			fromByte = byteIndices[len(returnMe)][0]
			toByte = byteIndices[len(returnMe)][1]
		}

		runePosition++
	}

	if fromRune != -1 {
		toRune := runePosition
		returnMe = append(returnMe, [2]int{fromRune, toRune})
	}

	return returnMe
}

// InRange says true if the index is part of a regexp match
func (mr *MatchRanges) InRange(index int) bool {
	if mr == nil {
		return false
	}

	for _, match := range mr.Matches {
		matchFirstIndex := match[0]
		matchLastIndex := match[1] - 1

		if index < matchFirstIndex {
			continue
		}

		if index > matchLastIndex {
			continue
		}

		return true
	}

	return false
}
