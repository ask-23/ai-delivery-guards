package checks

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ask-23/ai-delivery-guards/internal/report"
)

func severities(findings []report.Finding) map[string]report.Severity {
	out := map[string]report.Severity{}
	for _, f := range findings {
		out[f.Check] = f.Severity
	}
	return out
}

func firstMessage(findings []report.Finding) string {
	if len(findings) == 0 {
		return ""
	}
	return findings[0].Message
}

// --- happy / sad fixture coverage -----------------------------------------

func TestEvalCoverage_PassesOnGoodFixture(t *testing.T) {
	findings, err := EvalCoverage("../../testdata/good")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := severities(findings)["eval_coverage"]; got != report.Pass {
		t.Fatalf("eval_coverage = %s, want PASS", got)
	}
	if !strings.Contains(firstMessage(findings), "evals") {
		t.Errorf("expected message to cite evals directory, got %q", firstMessage(findings))
	}
}

func TestEvalCoverage_FailsOnBadFixture(t *testing.T) {
	findings, err := EvalCoverage("../../testdata/bad")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := severities(findings)["eval_coverage"]; got != report.Fail {
		t.Fatalf("eval_coverage = %s, want FAIL", got)
	}
	if !strings.Contains(firstMessage(findings), "app/model.go") {
		t.Errorf("expected message to cite offending file, got %q", firstMessage(findings))
	}
}

func TestRollbackMetadata_PassesOnGoodFixture(t *testing.T) {
	findings, err := RollbackMetadata("../../testdata/good")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := severities(findings)["rollback_metadata"]; got != report.Pass {
		t.Fatalf("rollback_metadata = %s, want PASS", got)
	}
}

func TestRollbackMetadata_FailsOnBadFixture(t *testing.T) {
	findings, err := RollbackMetadata("../../testdata/bad")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := severities(findings)["rollback_metadata"]; got != report.Fail {
		t.Fatalf("rollback_metadata = %s, want FAIL", got)
	}
}

func TestObservability_PassesOnGoodFixture(t *testing.T) {
	findings, err := Observability("../../testdata/good")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := severities(findings)["observability"]; got != report.Pass {
		t.Fatalf("observability = %s, want PASS", got)
	}
}

func TestObservability_FailsOnBadFixture(t *testing.T) {
	findings, err := Observability("../../testdata/bad")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := severities(findings)["observability"]; got != report.Fail {
		t.Fatalf("observability = %s, want FAIL", got)
	}
}

// --- false-positive regression cases --------------------------------------

// Codex H1 / H3: an observability marker in an unrelated docs file MUST NOT
// satisfy a call site that lacks the marker in its own file.
func TestObservability_DocsTraceIdDoesNotSatisfyCallSite(t *testing.T) {
	findings, err := Observability("../../testdata/false_positive_observability")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := severities(findings)["observability"]; got != report.Fail {
		t.Fatalf("observability = %s, want FAIL (docs trace_id must not count)", got)
	}
}

// Codex H1 / H3: the word "rollback" in a runbook MUST NOT satisfy a manifest
// that declares no rollback target.
func TestRollbackMetadata_DocsRollbackWordDoesNotSatisfyManifest(t *testing.T) {
	findings, err := RollbackMetadata("../../testdata/false_positive_rollback")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := severities(findings)["rollback_metadata"]; got != report.Fail {
		t.Fatalf("rollback_metadata = %s, want FAIL (docs 'rollback' must not count)", got)
	}
}

// Codex H3: when multiple LLM call sites exist, the observability check must
// flag the file that is uninstrumented even if others are covered.
func TestObservability_UninstrumentedSecondCallSiteFails(t *testing.T) {
	findings, err := Observability("../../testdata/multi_callsite")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := severities(findings)["observability"]; got != report.Fail {
		t.Fatalf("observability = %s, want FAIL (one uncovered call site)", got)
	}
	if !strings.Contains(firstMessage(findings), "scorer.go") {
		t.Errorf("expected message to cite scorer.go, got %q", firstMessage(findings))
	}
}

// --- edge cases ------------------------------------------------------------

func TestEvalCoverage_NoCallSitesPasses(t *testing.T) {
	findings, err := EvalCoverage("../../testdata/empty")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := severities(findings)["eval_coverage"]; got != report.Pass {
		t.Fatalf("eval_coverage on empty dir = %s, want PASS", got)
	}
}

func TestObservability_NoCallSitesPasses(t *testing.T) {
	findings, err := Observability("../../testdata/empty")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := severities(findings)["observability"]; got != report.Pass {
		t.Fatalf("observability on empty dir = %s, want PASS", got)
	}
}

func TestRollbackMetadata_NoManifestWarns(t *testing.T) {
	findings, err := RollbackMetadata("../../testdata/empty")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := severities(findings)["rollback_metadata"]; got != report.Warn {
		t.Fatalf("rollback_metadata on empty dir = %s, want WARN", got)
	}
}

// --- registry / config ----------------------------------------------------

func TestAll_ReturnsThreeChecks(t *testing.T) {
	if got := len(All()); got != 3 {
		t.Fatalf("All() returned %d checks, want 3", got)
	}
}

func TestDefaultConfig_EnablesAll(t *testing.T) {
	enabled := Enabled(DefaultConfig())
	if len(enabled) != 3 {
		t.Fatalf("default Enabled() returned %d checks, want 3", len(enabled))
	}
}

func TestLoadConfig_MissingFileReturnsDefaults(t *testing.T) {
	dir := t.TempDir()
	cfg, err := LoadConfig(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.RequireEvalCoverage || !cfg.RequireRollbackMetadata || !cfg.RequireObservability {
		t.Fatalf("missing config should default to all-enabled, got %+v", cfg)
	}
}

func TestLoadConfig_DisablesObservability(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".ai-guards.yml")
	if err := os.WriteFile(path, []byte("require_observability: false\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.RequireObservability {
		t.Fatalf("require_observability should be false, got %+v", cfg)
	}
	if !cfg.RequireEvalCoverage {
		t.Fatalf("require_eval_coverage should remain default true, got %+v", cfg)
	}
}

// Codex R2-2: inline comments must be stripped before boolean parsing,
// otherwise `true # comment` parses as junk and silently disables a guard.
func TestLoadConfig_InlineCommentStripped(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".ai-guards.yml")
	body := "require_eval_coverage: true # leave enabled\nrequire_observability: false # disabled for now\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.RequireEvalCoverage {
		t.Errorf("inline-commented true should still parse true, got %+v", cfg)
	}
	if cfg.RequireObservability {
		t.Errorf("inline-commented false should still parse false, got %+v", cfg)
	}
}

// Codex R2-2: a typo in a boolean must error out rather than silently
// disabling the guard.
func TestLoadConfig_TypoRejected(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".ai-guards.yml")
	if err := os.WriteFile(path, []byte("require_eval_coverage: ture\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadConfig(dir)
	if err == nil {
		t.Fatal("expected error for invalid boolean value, got nil")
	}
	if !strings.Contains(err.Error(), "ture") {
		t.Errorf("error should cite offending value, got %q", err.Error())
	}
}

// Codex R2-3: a README under deploy/ must NOT be treated as a deployment
// manifest.
func TestRollbackMetadata_ReadmeUnderDeployNotAManifest(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "deploy"), 0o755); err != nil {
		t.Fatal(err)
	}
	readme := "# Deploy\n\nRun the rollback procedure manually if needed.\n"
	if err := os.WriteFile(filepath.Join(dir, "deploy", "README.md"), []byte(readme), 0o644); err != nil {
		t.Fatal(err)
	}
	findings, err := RollbackMetadata(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := severities(findings)["rollback_metadata"]; got != report.Warn {
		t.Fatalf("rollback_metadata = %s, want WARN (README is not a manifest)", got)
	}
}

// Codex R2-3: a file named "manifesto.md" must not be treated as a deployment
// manifest just because its basename starts with "manifest".
func TestRollbackMetadata_ManifestoMdNotAManifest(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "docs", "manifesto.md"), []byte("our principles"), 0o644); err != nil {
		t.Fatal(err)
	}
	findings, err := RollbackMetadata(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := severities(findings)["rollback_metadata"]; got != report.Warn {
		t.Fatalf("rollback_metadata = %s, want WARN (manifesto.md is not a deployment manifest)", got)
	}
}

// Codex R2-1: a file named eval_helper.go must NOT satisfy eval_coverage.
// Only an actual directory named eval/evals/evaluation counts.
func TestEvalCoverage_EvalHelperFileDoesNotSatisfy(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "app"), 0o755); err != nil {
		t.Fatal(err)
	}
	// LLM call site
	if err := os.WriteFile(filepath.Join(dir, "app", "model.go"), []byte("// openai call\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Decoy: a file named eval_helper.go in app/ — basename contains "eval"
	// but the directory itself is "app", not an eval directory.
	if err := os.WriteFile(filepath.Join(dir, "app", "eval_helper.go"), []byte("package app\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	findings, err := EvalCoverage(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := severities(findings)["eval_coverage"]; got != report.Fail {
		t.Fatalf("eval_coverage = %s, want FAIL (eval_helper.go must not satisfy)", got)
	}
}
