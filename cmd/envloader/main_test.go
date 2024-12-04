package main

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetEnvAsJSONWithInclusion(t *testing.T) {
	// Set environment variables to test parsing
	t.Setenv("ADDRESS", "localhost:8080")
	t.Setenv("REPORT_INTERVAL", "10")
	t.Setenv("POLL_INTERVAL", "5")

	// Call the function to get environment variables as JSON
	result, err := getEnvAsJSON()
	require.NoError(t, err, "getEnvAsJSON should not return an error")

	// Parse the result into a map for inclusion check
	var resultMap map[string]string
	err = json.Unmarshal([]byte(result), &resultMap)
	require.NoError(t, err, "Unmarshalling the result JSON should not return an error")

	// Expected key-value pairs
	expected := map[string]string{
		"ADDRESS":         "localhost:8080",
		"REPORT_INTERVAL": "10",
		"POLL_INTERVAL":   "5",
	}

	// Assert that each expected key-value pair is present in the result
	for key, value := range expected {
		assert.Equal(t, value, resultMap[key], "Expected %s to be %s in the result JSON", key, value)
	}
}
