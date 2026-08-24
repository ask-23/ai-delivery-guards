package checks

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

var defaultExcludes = []string{
	".git",
	"vendor",
	"node_modules",
	"internal/checks",
	"testdata",
}

type fileMatch struct {
	path   string
	marker string
}

func isExcludedRelPath(relSlash string, excludes []string) bool {
	for _, ex := range excludes {
		if relSlash == ex || strings.HasPrefix(relSlash, ex+"/") || strings.Contains(relSlash, "/"+ex+"/") || strings.HasSuffix(relSlash, "/"+ex) {
			return true
		}
	}
	return false
}

func walkFiles(root string, excludes []string, fn func(path string, content string) error) error {
	if _, _, err := canonicalRootPaths(root); err != nil {
		return err
	}

	skip := append([]string{}, defaultExcludes...)
	skip = append(skip, excludes...)
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return fmt.Errorf("resolve path %q relative to scan root: %w", path, relErr)
		}
		relSlash := filepath.ToSlash(rel)
		if relSlash == "." {
			return nil
		}
		if isExcludedRelPath(relSlash, skip) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink is not allowed in scan root: %s", relSlash)
		}
		if d.IsDir() {
			return nil
		}
		b, readErr := os.ReadFile(path)
		if readErr != nil {
			return fmt.Errorf("read %s: %w", relSlash, readErr)
		}
		return fn(path, string(b))
	})
}

func findMatches(root string, needles []string, excludes ...string) ([]fileMatch, error) {
	var matches []fileMatch
	err := walkFiles(root, excludes, func(path, content string) error {
		lower := strings.ToLower(content)
		for _, n := range needles {
			if strings.Contains(lower, strings.ToLower(n)) {
				matches = append(matches, fileMatch{path: path, marker: n})
				return nil
			}
		}
		return nil
	})
	return matches, err
}

func hasDirectoryNamed(root string, names []string, excludes ...string) (bool, string, error) {
	if _, _, err := canonicalRootPaths(root); err != nil {
		return false, "", err
	}

	skip := append([]string{}, defaultExcludes...)
	skip = append(skip, excludes...)
	wanted := make(map[string]struct{}, len(names))
	for _, n := range names {
		wanted[strings.ToLower(n)] = struct{}{}
	}
	var foundPath string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return fmt.Errorf("resolve path %q relative to scan root: %w", path, relErr)
		}
		relSlash := filepath.ToSlash(rel)
		if relSlash == "." {
			return nil
		}
		if isExcludedRelPath(relSlash, skip) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink is not allowed in scan root: %s", relSlash)
		}
		if !d.IsDir() {
			return nil
		}
		if _, ok := wanted[strings.ToLower(d.Name())]; ok {
			foundPath = path
			return fs.SkipAll
		}
		return nil
	})
	return foundPath != "", foundPath, err
}

func relOrAbs(root, path string) string {
	if r, err := filepath.Rel(root, path); err == nil {
		return r
	}
	return path
}
