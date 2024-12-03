package testutils_test

import (
	"testing"

	testutils "github.com/npavlov/go-loyalty-service/internal/test_utils"
	"github.com/npavlov/go-loyalty-service/internal/utils"
	"github.com/stretchr/testify/assert"
)

func TestGenerateLuhnNumber(t *testing.T) {
	t.Run("Generates number of correct length", func(t *testing.T) {
		length := 10
		result := testutils.GenerateLuhnNumber(length)
		assert.Equal(t, length, len(result), "Generated number should have the specified length")
	})

	t.Run("Generated number is Luhn-valid", func(t *testing.T) {
		length := 16
		result := testutils.GenerateLuhnNumber(length)

		isValid := utils.LuhnCheck(result)
		assert.True(t, isValid, "Generated number should be Luhn valid")
	})

	t.Run("Panics on invalid length", func(t *testing.T) {
		assert.Panics(t, func() {
			testutils.GenerateLuhnNumber(1)
		}, "Should panic for length less than 2")
	})
}
