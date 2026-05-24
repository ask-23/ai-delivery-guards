package checks

import (
	"fmt"
	"os"
	"strings"

	"github.com/ask-23/ai-delivery-guards/internal/report"
)

var observabilityMarkers = []string{"trace_id", "span", "token_usage", "model_latency", "llm_latency"}

func Observability(root string) ([]report.Finding, error) {
	callSites, err := findMatches(root, llmMarkers)
	if err != nil {
		return nil, err
	}
	if len(callSites) == 0 {
		return []report.Finding{{
			Severity: report.Pass,
			Check:    "observability",
			Message:  "no LLM call sites detected",
		}}, nil
	}
	// Each call site must carry an observability marker IN THE SAME FILE.
	// A trace_id buried in an unrelated README does not prove the call is instrumented.
	for _, cs := range callSites {
		if !fileContainsAny(cs.path, observabilityMarkers) {
			return []report.Finding{{
				Severity: report.Fail,
				Check:    "observability",
				Message: fmt.Sprintf(
					"LLM call site at %s (marker %q) has no trace/metric marker in the same file; expected one of %v",
					relOrAbs(root, cs.path), cs.marker, observabilityMarkers,
				),
			}}, nil
		}
	}
	first := callSites[0]
	return []report.Finding{{
		Severity: report.Pass,
		Check:    "observability",
		Message: fmt.Sprintf(
			"%d LLM call site(s) checked; observability marker present (e.g. %s)",
			len(callSites), relOrAbs(root, first.path),
		),
	}}, nil
}

func fileContainsAny(path string, needles []string) bool {
	b, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	lower := strings.ToLower(string(b))
	for _, n := range needles {
		if strings.Contains(lower, strings.ToLower(n)) {
			return true
		}
	}
	return false
}
