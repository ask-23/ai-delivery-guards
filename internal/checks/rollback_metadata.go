package checks

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ask-23/ai-delivery-guards/internal/report"
)

var rollbackMarkers = []string{"rollback", "fallback_model", "previous_model"}

var manifestExts = map[string]struct{}{
	".yml":      {},
	".yaml":     {},
	".json":     {},
	".tf":       {},
	".toml":     {},
	".hcl":      {},
	".template": {},
}

func RollbackMetadata(root string) ([]report.Finding, error) {
	manifests, err := findDeploymentManifests(root)
	if err != nil {
		return nil, err
	}
	if len(manifests) == 0 {
		return []report.Finding{{
			Severity: report.Warn,
			Check:    "rollback_metadata",
			Message:  "no deployment manifest found (looked for manifest.* files or *.yml/*.yaml/*.json/*.tf/*.toml under deploy/ or deployment/)",
		}}, nil
	}
	for _, m := range manifests {
		if !fileContainsAny(m, rollbackMarkers) {
			return []report.Finding{{
				Severity: report.Fail,
				Check:    "rollback_metadata",
				Message: fmt.Sprintf(
					"deployment manifest %s declares no rollback target; expected one of %v",
					relOrAbs(root, m), rollbackMarkers,
				),
			}}, nil
		}
	}
	return []report.Finding{{
		Severity: report.Pass,
		Check:    "rollback_metadata",
		Message: fmt.Sprintf(
			"%d deployment manifest(s) checked; rollback declaration present (e.g. %s)",
			len(manifests), relOrAbs(root, manifests[0]),
		),
	}}, nil
}

func findDeploymentManifests(root string) ([]string, error) {
	var out []string
	err := walkFiles(root, nil, func(path, _ string) error {
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return fmt.Errorf("resolve deployment path %q relative to scan root: %w", path, err)
		}
		ext := strings.ToLower(filepath.Ext(path))
		base := strings.ToLower(filepath.Base(path))
		baseNoExt := strings.TrimSuffix(base, ext)

		_, extOK := manifestExts[ext]
		underDeploy := hasDirectoryComponent(rel, "deploy", "deployment")

		switch {
		case strings.HasPrefix(baseNoExt, "manifest") && extOK:
			out = append(out, path)
		case underDeploy && extOK:
			out = append(out, path)
		}
		return nil
	})
	return out, err
}

func hasDirectoryComponent(rel string, names ...string) bool {
	wanted := make(map[string]struct{}, len(names))
	for _, name := range names {
		wanted[strings.ToLower(name)] = struct{}{}
	}

	dir := filepath.ToSlash(filepath.Dir(rel))
	if dir == "." {
		return false
	}
	for _, component := range strings.Split(dir, "/") {
		if _, ok := wanted[strings.ToLower(component)]; ok {
			return true
		}
	}
	return false
}
