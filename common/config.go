package common

import (
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

// LoadConfig loads configuration from environment
func LoadConfig() (*Config, error) {
	// Load .env file if it exists
	godotenv.Load()

	config := &Config{
		CredentialsPath: getEnvOrDefault("GMAIL_CREDENTIALS_PATH", "gmail.json"),
		TokenDir:        getEnvOrDefault("TOKEN_DIR", getDefaultTokenDir()),
		UserEmail:       getEnvOrDefault("USER_EMAIL", "help@blue.cc"),
	}

	return config, nil
}

// getEnvOrDefault returns env value or default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getDefaultTokenDir returns the default token directory
func getDefaultTokenDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".support-agent")
}