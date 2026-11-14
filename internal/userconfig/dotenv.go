package userconfig

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

// credentialVariables are the only variables a .env file may supply. It must
// not select programs to run (VISUAL, EDITOR) or redirect requests
// (MEANS_EMBEDDING_BASE_URL).
var credentialVariables = []string{"OPENAI_API_KEY", "OPENROUTER_API_KEY", "MEANS_EMBEDDING_API_KEY"}

// LoadEnvironment fills unset credential variables from the working directory's
// .env. Collections may be untrusted, so a .env inside a collection is never
// read: the file is skipped when the working directory or one of its parents
// contains MEANS.md. Other variables in the file are ignored. Existing
// variables, even empty ones, win. A missing file is optional. Parse errors
// deliberately omit credential values.
func LoadEnvironment() error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("locate working directory for .env: %w", err)
	}
	for dir := cwd; ; dir = filepath.Dir(dir) {
		if _, err := os.Lstat(filepath.Join(dir, "MEANS.md")); err == nil {
			return nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("check %s for a collection: %w", dir, err)
		}
		if dir == filepath.Dir(dir) {
			break
		}
	}
	filename := filepath.Join(cwd, ".env")
	source, err := os.ReadFile(filename)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read .env file %s: %w", filename, err)
	}
	parsed, err := godotenv.Unmarshal(string(source))
	if err != nil {
		return fmt.Errorf("invalid .env file %s", filename)
	}
	for _, key := range credentialVariables {
		value, found := parsed[key]
		if _, exists := os.LookupEnv(key); found && !exists {
			if err := os.Setenv(key, value); err != nil {
				return fmt.Errorf("set variable from .env: %w", err)
			}
		}
	}
	return nil
}
