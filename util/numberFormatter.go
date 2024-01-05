package util

import "fmt"

// Formats a number into a string with _ between each three-group of digits, for
// numbers >= 10_000.
//
// Regarding the >= 10_000 exception:
// https://en.wikipedia.org/wiki/Decimal_separator#Exceptions_to_digit_grouping
func FormatNumber(number uint) string {
	if number < 10_000 {
		return fmt.Sprint(number)
	}

	result := ""

	chars := []rune(fmt.Sprint(number))
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
