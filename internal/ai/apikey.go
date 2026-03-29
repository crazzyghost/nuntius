package ai

import (
	"fmt"
	"os"
)

// APIKeyEnvVar is the canonical environment variable for the nuntius API key.
const APIKeyEnvVar = "NUNTIUS_AI_API_KEY"

// RequiresAPIKey reports whether the given provider and mode combination
// requires an API key. CLI-mode providers and Ollama (local) do not.
func RequiresAPIKey(provider string, mode ProviderMode) bool {
	if mode == ModeCLI {
		return false
	}
	// Ollama runs locally — no API key needed even in API mode.
	return provider != "ollama"
}

// ResolveAPIKey reads the NUNTIUS_AI_API_KEY environment variable and returns
// its value. If the variable is empty or unset, it returns a descriptive
// error naming both the provider and the required env var.
func ResolveAPIKey(provider string) (string, error) {
	key := os.Getenv(APIKeyEnvVar)
	if key == "" {
		return "", fmt.Errorf(
			"%s: API key not set — export %s=<your-key>",
			provider, APIKeyEnvVar,
		)
	}
	return key, nil
}
