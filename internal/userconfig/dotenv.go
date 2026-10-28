package userconfig

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

// LoadEnvironment fills unset environment variables from the collection's .env,
// then the working directory's .env. Existing variables, even empty ones, win.
// Missing files are optional. Parse errors deliberately omit credential values.
func LoadEnvironment(path string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("locate working directory for .env: %w", err)
	}
	collection, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolve collection for .env: %w", err)
	}
	paths := []string{filepath.Join(collection, ".env")}
	if collection != cwd {
		paths = append(paths, filepath.Join(cwd, ".env"))
	}
	values := make(map[string]string)
	for _, filename := range paths {
		source, err := os.ReadFile(filename)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return fmt.Errorf("read .env file %s: %w", filename, err)
		}
		parsed, err := godotenv.Unmarshal(string(source))
		if err != nil {
			return fmt.Errorf("invalid .env file %s", filename)
		}
		for key, value := range parsed {
			if _, exists := values[key]; !exists {
				values[key] = value
			}
		}
	}
	for key, value := range values {
		if _, exists := os.LookupEnv(key); !exists {
			if err := os.Setenv(key, value); err != nil {
				return fmt.Errorf("set variable from .env: %w", err)
			}
		}
	}
	return nil
}
