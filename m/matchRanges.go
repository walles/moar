package m

import "regexp"

// MatchRanges collects match indices
type MatchRanges struct {
	Matches [][2]int
}

// getMatchRanges locates one or more regexp matches in a string
func getMatchRanges(String *string, Pattern *regexp.Regexp) *MatchRanges {
	if Pattern == nil {
		return nil
	}

	return &MatchRanges{
		Matches: toRunePositions(Pattern.FindAllStringIndex(*String, -1), String),
	}
}

// Convert byte indices to rune indices
func toRunePositions(byteIndices [][]int, matchedString *string) [][2]int {
	var returnMe [][2]int
	if len(byteIndices) == 0 {
		// Nothing to see here, move along
		return returnMe
	}

	runeIndex := 0
	byteIndicesToRuneIndices := make(map[int]int, 0)
	for byteIndex := range *matchedString {
		byteIndicesToRuneIndices[byteIndex] = runeIndex

		runeIndex++
	}

	// If a match touches the end of the string, that will be encoded as one
	// byte past the end of the string. Therefore we must add a mapping for
	// first-index-after-the-end.
	byteIndicesToRuneIndices[len(*matchedString)] = runeIndex

	for _, bytePair := range byteIndices {
		fromRuneIndex := byteIndicesToRuneIndices[bytePair[0]]
		toRuneIndex := byteIndicesToRuneIndices[bytePair[1]]
		returnMe = append(returnMe, [2]int{fromRuneIndex, toRuneIndex})
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
