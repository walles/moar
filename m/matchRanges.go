package m

import "regexp"

// MatchRanges collects match indices
type MatchRanges struct {
	Matches [][]int
}

// GetMatchRanges locates a regexp in a string
func GetMatchRanges(String string, Pattern *regexp.Regexp) *MatchRanges {
	return &MatchRanges{
		Matches: Pattern.FindAllStringIndex(String, -1),
	}
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
