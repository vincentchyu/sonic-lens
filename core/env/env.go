package env

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

func init() {
	// Try to load .env file from current directory or scripts directory
	envPaths := []string{
		".env",
		"scripts/.env",
	}

	var err error
	for _, path := range envPaths {
		err = godotenv.Load(path)
		if err == nil {
			log.Printf("✓ Loaded .env from: %s\n", path)
			break
		}
	}

	if err != nil {
		panic(err)
	}
}

// GetEnv tries multiple environment variable names and returns the first non-empty value
func GetEnv(names ...string) string {
	for _, name := range names {
		if value := os.Getenv(name); value != "" {
			return value
		}
	}
	return ""
}
