package testutils_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	testutils "github.com/npavlov/go-loyalty-service/internal/test_utils"
	"github.com/npavlov/go-loyalty-service/internal/utils"
)

func TestGenerateLuhnNumber(t *testing.T) {
	t.Parallel()

	t.Run("Generates number of correct length", func(t *testing.T) {
		t.Parallel()

		length := 10
		result := testutils.GenerateLuhnNumber(length)
		assert.Len(t, result, length, "Generated number should have the specified length")
	})

	t.Run("Generated number is Luhn-valid", func(t *testing.T) {
		t.Parallel()

		length := 16
		result := testutils.GenerateLuhnNumber(length)

		isValid := utils.LuhnCheck(result)
		assert.True(t, isValid, "Generated number should be Luhn valid")
	})

	t.Run("Panics on invalid length", func(t *testing.T) {
		t.Parallel()

		assert.Panics(t, func() {
			testutils.GenerateLuhnNumber(1)
		}, "Should panic for length less than 2")
	})
}
