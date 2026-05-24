package checks

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config mirrors .ai-guards.yml. Missing keys default to true so a missing or
// blank file leaves the full check set enabled.
type Config struct {
	RequireEvalCoverage     bool
	RequireRollbackMetadata bool
	RequireObservability    bool
}

func DefaultConfig() Config {
	return Config{
		RequireEvalCoverage:     true,
		RequireRollbackMetadata: true,
		RequireObservability:    true,
	}
}

// LoadConfig reads a .ai-guards.yml file at <root>/.ai-guards.yml. The format
// is intentionally minimal: `key: true|false` per line, comments start with `#`
// and may appear inline after a value.
//
// Failure mode is deliberate: an unrecognized boolean value (typo, junk text)
// is treated as an ERROR rather than silently defaulting to false. A guard
// policy that mis-spells `true` should not silently disable a check.
func LoadConfig(root string) (Config, error) {
	cfg := DefaultConfig()
	path := filepath.Join(root, ".ai-guards.yml")
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}
	for n, raw := range strings.Split(string(b), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Drop inline comment before further processing.
		if i := strings.Index(line, "#"); i >= 0 {
			line = strings.TrimSpace(line[:i])
		}
		key, val, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(strings.ToLower(key))
		val = strings.TrimSpace(strings.ToLower(val))
		// Strip optional surrounding quotes.
		val = strings.Trim(val, "\"'")

		enabled, parseErr := parseBool(val)
		if parseErr != nil {
			return cfg, fmt.Errorf(".ai-guards.yml:%d: %w", n+1, parseErr)
		}

		switch key {
		case "require_eval_coverage":
			cfg.RequireEvalCoverage = enabled
		case "require_rollback_metadata":
			cfg.RequireRollbackMetadata = enabled
		case "require_observability":
			cfg.RequireObservability = enabled
		default:
			// Unknown key is tolerated (forward-compatible); skip silently.
		}
	}
	return cfg, nil
}

func parseBool(s string) (bool, error) {
	switch s {
	case "true", "yes", "on", "1":
		return true, nil
	case "false", "no", "off", "0":
		return false, nil
	}
	return false, fmt.Errorf("invalid boolean %q (expected true|false|yes|no|on|off|1|0)", s)
}
