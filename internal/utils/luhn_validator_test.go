package utils_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	testutils "github.com/npavlov/go-loyalty-service/internal/test_utils"
	"github.com/npavlov/go-loyalty-service/internal/utils"
)

func TestLuhnCheck(t *testing.T) {
	t.Parallel()

	// Test valid Luhn numbers
	validNumbers := []string{
		testutils.GenerateLuhnNumber(16), // Randomly generated valid numbers
		testutils.GenerateLuhnNumber(15),
		testutils.GenerateLuhnNumber(12),
	}
	for _, number := range validNumbers {
		assert.True(t, utils.LuhnCheck(number), "Expected number %s to be valid", number)
	}

	// Test invalid Luhn numbers
	invalidNumbers := []string{
		"1234567890123456", // Arbitrary invalid numbers
		"987654321098765",
		"1111111111111111",
	}
	for _, number := range invalidNumbers {
		assert.False(t, utils.LuhnCheck(number), "Expected number %s to be invalid", number)
	}
}

func TestGenerateLuhnNumber(t *testing.T) {
	t.Parallel()

	// Generate numbers and ensure they pass the Luhn check
	for length := 10; length <= 20; length++ {
		number := testutils.GenerateLuhnNumber(length)
		assert.Equal(t, len(number), length, "Expected length of %d, got %d", length, len(number))
		assert.True(t, utils.LuhnCheck(number), "Generated number %s should be valid", number)
	}
}
