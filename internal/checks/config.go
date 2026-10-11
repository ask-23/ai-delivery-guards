package checks

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config controls which checks run. All checks are enabled by default. A
// policy can disable a subset only when it is supplied explicitly from outside
// the scan root.
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

// LoadPolicy returns the all-enabled default when policyPath is empty. When a
// policy is supplied, it must be a regular, non-symlink file outside root. This
// keeps the target repository from weakening the checks that evaluate it.
func LoadPolicy(root, policyPath string) (Config, error) {
	cfg := DefaultConfig()
	if strings.TrimSpace(policyPath) == "" {
		return cfg, nil
	}

	rootAbs, rootResolved, err := canonicalRootPaths(root)
	if err != nil {
		return cfg, err
	}

	policyAbs, err := filepath.Abs(policyPath)
	if err != nil {
		return cfg, fmt.Errorf("resolve policy path: %w", err)
	}
	policyAbs = filepath.Clean(policyAbs)

	info, err := os.Lstat(policyAbs)
	if err != nil {
		return cfg, fmt.Errorf("stat policy file %q: %w", policyPath, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return cfg, fmt.Errorf("policy file must not be a symlink: %s", policyAbs)
	}
	if !info.Mode().IsRegular() {
		return cfg, fmt.Errorf("policy path must be a regular file: %s", policyAbs)
	}

	if pathWithin(rootAbs, policyAbs) {
		return cfg, fmt.Errorf("policy file must be outside scan root: %s", policyAbs)
	}

	policyResolved, err := filepath.EvalSymlinks(policyAbs)
	if err != nil {
		return cfg, fmt.Errorf("resolve policy file %q: %w", policyPath, err)
	}
	policyResolved = filepath.Clean(policyResolved)
	if pathWithin(rootResolved, policyResolved) {
		return cfg, fmt.Errorf("policy file resolves inside scan root: %s", policyResolved)
	}

	cfg, err = loadConfigFile(policyResolved)
	if err != nil {
		return cfg, err
	}
	if !cfg.RequireEvalCoverage && !cfg.RequireRollbackMetadata && !cfg.RequireObservability {
		return cfg, fmt.Errorf("policy disables all checks")
	}
	return cfg, nil
}

func canonicalRootPaths(root string) (string, string, error) {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", "", fmt.Errorf("resolve scan root: %w", err)
	}
	rootAbs = filepath.Clean(rootAbs)
	info, err := os.Lstat(rootAbs)
	if err != nil {
		return "", "", fmt.Errorf("stat scan root %q: %w", root, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", "", fmt.Errorf("scan root must not be a symlink: %s", rootAbs)
	}
	if !info.IsDir() {
		return "", "", fmt.Errorf("scan root must be a directory: %s", rootAbs)
	}
	rootResolved, err := filepath.EvalSymlinks(rootAbs)
	if err != nil {
		return "", "", fmt.Errorf("resolve scan root %q: %w", root, err)
	}
	return rootAbs, filepath.Clean(rootResolved), nil
}

func pathWithin(root, candidate string) bool {
	rel, err := filepath.Rel(root, candidate)
	if err != nil {
		return false
	}
	rel = filepath.Clean(rel)
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)))
}

func loadConfigFile(path string) (Config, error) {
	cfg := DefaultConfig()
	b, err := os.ReadFile(path)
	if err != nil {
		return cfg, fmt.Errorf("read policy file %q: %w", path, err)
	}
	for n, raw := range strings.Split(string(b), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if i := strings.Index(line, "#"); i >= 0 {
			line = strings.TrimSpace(line[:i])
		}
		key, val, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(strings.ToLower(key))
		val = strings.TrimSpace(strings.ToLower(val))
		val = strings.Trim(val, "\"'")

		enabled, parseErr := parseBool(val)
		if parseErr != nil {
			return cfg, fmt.Errorf("%s:%d: %w", path, n+1, parseErr)
		}

		switch key {
		case "require_eval_coverage":
			cfg.RequireEvalCoverage = enabled
		case "require_rollback_metadata":
			cfg.RequireRollbackMetadata = enabled
		case "require_observability":
			cfg.RequireObservability = enabled
		default:
			// Unknown keys are forward-compatible and cannot weaken a default.
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
