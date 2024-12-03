package utils

import (
	"strconv"
	"strings"
)

const (
	DoubleThreshold = 9  // Threshold for subtracting
	Modulus         = 10 // Modulus for Luhn algorithm
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
			if digit > DoubleThreshold {
				digit -= DoubleThreshold // Subtract threshold if the result is greater
			}
		}
		sum += digit
	}

	// Check if the sum is divisible by the modulus
	return sum%Modulus == 0
}

// Reverse a string.
func reverseString(s string) string {
	var reversed strings.Builder
	for i := len(s) - 1; i >= 0; i-- {
		reversed.WriteByte(s[i])
	}

	return reversed.String()
}
