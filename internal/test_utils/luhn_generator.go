package testutils

import (
	"fmt"
	"math/rand"
	"time"
)

// GenerateLuhnNumber generates a Luhn-valid number of the specified length
func GenerateLuhnNumber(length int) string {
	if length <= 1 {
		panic("Length must be greater than 1")
	}

	// Generate random base number
	base := make([]int, length-1)
	rand.Seed(time.Now().UnixNano())
	for i := range base {
		base[i] = rand.Intn(10)
	}

	// Calculate the check digit
	checkDigit := calculateLuhnChecksum(base)

	// Combine base and check digit
	luhnNumber := append(base, checkDigit)
	result := ""
	for _, digit := range luhnNumber {
		result += fmt.Sprintf("%d", digit)
	}
	return result
}

// calculateLuhnChecksum calculates the Luhn check digit for a given number
func calculateLuhnChecksum(digits []int) int {
	sum := 0
	isSecond := true
	// Start from the rightmost digit and apply the Luhn algorithm
	for i := len(digits) - 1; i >= 0; i-- {
		d := digits[i]
		if isSecond {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		sum += d
		isSecond = !isSecond
	}
	// Calculate the check digit
	return (10 - (sum % 10)) % 10
}
