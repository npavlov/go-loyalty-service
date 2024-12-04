package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

func main() {
	envJSON, err := getEnvAsJSON()
	if err != nil {
		log.Error().Err(err).Msg("Error marshalling envs to JSON")

		return
	}

	// Output the JSON string
	//nolint:forbidigo
	fmt.Println(envJSON)
}

// getEnvAsJSON retrieves environment variables as a JSON string.
func getEnvAsJSON() (string, error) {
	envs := make(map[string]string)
	for _, env := range os.Environ() {
		pair := splitEnv(env)
		envs[pair[0]] = pair[1]
	}

	envJSON, err := json.Marshal(envs)
	if err != nil {
		return "", errors.Wrap(err, "Error marshalling envs to JSON")
	}

	return string(envJSON), nil
}

// splitEnv splits an environment variable into key and value parts.
func splitEnv(env string) []string {
	for i := range env {
		if env[i] == '=' {
			return []string{env[:i], env[i+1:]}
		}
	}

	return []string{env, ""}
}
