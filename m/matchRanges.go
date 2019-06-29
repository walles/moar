package m

import "regexp"

type MatchRanges struct{}

// GetMatchRanges locates a regexp in a string
func GetMatchRanges(String string, Pattern *regexp.Regexp) *MatchRanges {
	// FIXME: Actually create an object
	return nil
}

// InRange says true if the index is part of a regexp match
func (mr *MatchRanges) InRange(index int) bool {
	if mr == nil {
		return false
	}

	// FIXME: Actually check
	return false
}
