package testutils

import (
	"fmt"
	"math/rand"
	"strconv"
	"time"
)

const (
	// Constants for Luhn algorithm.
	minLength            = 1  // Minimum allowed length for the Luhn number
	checkDigitBase       = 10 // Base for calculating the checksum
	checkDigitAdjustment = 9  // Adjustment value when the doubled digit exceeds 9
)

// GenerateLuhnNumber generates a Luhn-valid number of the specified length.
func GenerateLuhnNumber(length int) string {
	if length <= minLength {
		panic(fmt.Sprintf("Length must be greater than %d", minLength))
	}

	// Generate random base number
	base := make([]int, length-1)

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := range base {
		base[i] = rng.Intn(checkDigitBase)
	}

	// Calculate the check digit
	checkDigit := calculateLuhnChecksum(base)

	// Combine base and check digit
	luhnNumber := append(base, checkDigit)
	result := ""
	for _, digit := range luhnNumber {
		result += strconv.Itoa(digit)
	}

	return result
}

// calculateLuhnChecksum calculates the Luhn check digit for a given number.
func calculateLuhnChecksum(digits []int) int {
	sum := 0
	isSecond := true
	// Start from the rightmost digit and apply the Luhn algorithm
	for i := len(digits) - 1; i >= 0; i-- {
		digit := digits[i]
		if isSecond {
			digit *= 2
			if digit > checkDigitBase-1 { // If doubled digit exceeds 9
				digit -= checkDigitAdjustment
			}
		}
		sum += digit
		isSecond = !isSecond
	}
	// Calculate the check digit
	return (checkDigitBase - (sum % checkDigitBase)) % checkDigitBase
}
