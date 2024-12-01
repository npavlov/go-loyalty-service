package utils

import (
	"strconv"
	"strings"
)

// LuhnCheck checks if the number is valid according to the Luhn algorithm.
func LuhnCheck(number string) bool {
	// Reverse the number for easier calculation from right to left
	reversedNumber := reverseString(number)
	sum := 0

	// Iterate over the digits
	for i, digitRune := range reversedNumber {
		digit, _ := strconv.Atoi(string(digitRune))

		// Double every second digit
		if i%2 != 0 {
			digit *= 2
			if digit > 9 {
				digit -= 9 // Subtract 9 if the result is greater than 9
			}
		}
		sum += digit
	}

	// Check if the sum is divisible by 10
	return sum%10 == 0
}

// Reverse a string.
func reverseString(s string) string {
	var reversed strings.Builder
	for i := len(s) - 1; i >= 0; i-- {
		reversed.WriteByte(s[i])
	}
	return reversed.String()
}
