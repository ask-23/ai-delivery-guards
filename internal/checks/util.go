package checks

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// defaultExcludes are path fragments skipped by every scan.
// The guard's own internal/, testdata/, examples/, and policy files contain marker
// strings ("openai", "rollback", "trace_id") for demonstration purposes. Without
// these exclusions a scan of the guard repo against itself would self-satisfy.
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

// isExcludedRelPath reports whether a forward-slash relative path matches any
// excluded fragment by exact directory component or prefix.
func isExcludedRelPath(relSlash string, excludes []string) bool {
	for _, ex := range excludes {
		if relSlash == ex || strings.HasPrefix(relSlash, ex+"/") || strings.Contains(relSlash, "/"+ex+"/") || strings.HasSuffix(relSlash, "/"+ex) {
			return true
		}
	}
	return false
}

func walkFiles(root string, excludes []string, fn func(path string, content string) error) error {
	skip := append([]string{}, defaultExcludes...)
	skip = append(skip, excludes...)
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
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
		if d.IsDir() {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return nil
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

// hasDirectoryNamed returns the first directory whose own basename matches one
// of the supplied names (case-insensitive). It walks relative paths only and
// honors the default + caller-supplied exclusions, so a parent directory the
// guard happens to live under cannot accidentally satisfy a check.
func hasDirectoryNamed(root string, names []string, excludes ...string) (bool, string, error) {
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
		if !d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		relSlash := filepath.ToSlash(rel)
		if relSlash == "." {
			return nil
		}
		if isExcludedRelPath(relSlash, skip) {
			return filepath.SkipDir
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
