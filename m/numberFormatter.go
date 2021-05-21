package m

import "fmt"

// Formats a number into a string with _ between each three-group of digits
func formatNumber(number uint) string {
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
