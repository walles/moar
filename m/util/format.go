package util

import "fmt"

// Formats a positive number into a string with _ between each three-group of
// digits, for numbers >= 10_000.
//
// Regarding the >= 10_000 exception:
// https://en.wikipedia.org/wiki/Decimal_separator#Exceptions_to_digit_grouping
func FormatInt(i int) string {
	if i < 10_000 {
		return fmt.Sprint(i)
	}

	result := ""

	chars := []rune(fmt.Sprint(i))
	addCount := 0
	for i := len(chars) - 1; i >= 0; i-- {
		char := chars[i]
		if len(result) > 0 && addCount%3 == 0 {
			result = "_" + result
		}
		result = string(char) + result
		addCount++
	}

	return result
}
