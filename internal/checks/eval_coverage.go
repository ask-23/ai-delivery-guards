package checks

import (
	"fmt"

	"github.com/ask-23/ai-delivery-guards/internal/report"
)

var llmMarkers = []string{"openai", "anthropic", "llm.call", "model.invoke"}

// evalDirNames are the directory basenames that count as eval evidence. Match
// is exact (case-insensitive), not substring, so a file named `eval_helper.go`
// or a parent directory named `evaluation_docs` does not accidentally satisfy.
var evalDirNames = []string{"eval", "evals", "evaluation", "evaluations"}

func EvalCoverage(root string) ([]report.Finding, error) {
	callSites, err := findMatches(root, llmMarkers)
	if err != nil {
		return nil, err
	}
	if len(callSites) == 0 {
		return []report.Finding{{
			Severity: report.Pass,
			Check:    "eval_coverage",
			Message:  "no LLM call sites detected",
		}}, nil
	}
	hasEval, evalPath, err := hasDirectoryNamed(root, evalDirNames)
	if err != nil {
		return nil, err
	}
	if !hasEval {
		first := callSites[0]
		return []report.Finding{{
			Severity: report.Fail,
			Check:    "eval_coverage",
			Message: fmt.Sprintf(
				"LLM call site at %s (marker %q) has no eval coverage; create a directory named one of %v",
				relOrAbs(root, first.path), first.marker, evalDirNames,
			),
		}}, nil
	}
	first := callSites[0]
	return []report.Finding{{
		Severity: report.Pass,
		Check:    "eval_coverage",
		Message: fmt.Sprintf(
			"LLM call site at %s covered by eval directory %s",
			relOrAbs(root, first.path), relOrAbs(root, evalPath),
		),
	}}, nil
}
