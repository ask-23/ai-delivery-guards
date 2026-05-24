package checks

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ask-23/ai-delivery-guards/internal/report"
)

var rollbackMarkers = []string{"rollback", "fallback_model", "previous_model"}

// manifestExts are the extensions we accept as deployment-manifest content.
// A README in deploy/ is not a manifest; a manifest.yml is.
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

// findDeploymentManifests accepts a file as a deployment manifest only when:
//
//   - its basename (lowercased, before extension) starts with "manifest", OR
//   - it sits under a /deploy/ or /deployment/ directory AND has a recognized
//     manifest extension (yml/yaml/json/tf/toml/hcl/template).
//
// This excludes readmes, runbooks, and unrelated files that merely happen to
// live near a deployment directory or contain the word "manifest" in prose.
func findDeploymentManifests(root string) ([]string, error) {
	var out []string
	err := walkFiles(root, nil, func(path, _ string) error {
		relSlash := filepath.ToSlash(path)
		lower := strings.ToLower(relSlash)
		ext := strings.ToLower(filepath.Ext(path))
		base := strings.ToLower(filepath.Base(path))
		baseNoExt := strings.TrimSuffix(base, ext)

		_, extOK := manifestExts[ext]
		underDeploy := strings.Contains(lower, "/deploy/") || strings.Contains(lower, "/deployment/")

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
