package checks

import "github.com/ask-23/ai-delivery-guards/internal/report"

type CheckFunc func(root string) ([]report.Finding, error)

// All returns every check. The CLI applies config.toggle filtering on top.
func All() []CheckFunc {
	return []CheckFunc{EvalCoverage, RollbackMetadata, Observability}
}

// Enabled returns the subset of checks the config has not disabled.
// Used by the CLI so `.ai-guards.yml` actually controls behavior.
func Enabled(cfg Config) []CheckFunc {
	var out []CheckFunc
	if cfg.RequireEvalCoverage {
		out = append(out, EvalCoverage)
	}
	if cfg.RequireRollbackMetadata {
		out = append(out, RollbackMetadata)
	}
	if cfg.RequireObservability {
		out = append(out, Observability)
	}
	return out
}
