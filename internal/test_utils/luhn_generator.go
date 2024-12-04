package testutils

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strconv"
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

	for i := range base {
		randVal, _ := rand.Int(rand.Reader, big.NewInt(checkDigitBase))
		base[i] = int(randVal.Int64())
	}

	// Calculate the check digit
	checkDigit := calculateLuhnChecksum(base)

	// Combine base and check digit
	//nolint:gocritic,makezero
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
