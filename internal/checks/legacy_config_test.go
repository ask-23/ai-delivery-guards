package checks

import (
	"os"
	"path/filepath"
)

// LoadConfig is a test-only adapter retained for the original parser regression
// tests. Production code cannot load policy from the repository being scanned.
func LoadConfig(root string) (Config, error) {
	path := filepath.Join(root, ".ai-guards.yml")
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return DefaultConfig(), err
	}
	return loadConfigFile(path)
}
