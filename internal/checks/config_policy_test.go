package checks

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func trustedPolicyFixture(t *testing.T, body string) (string, string) {
	t.Helper()
	base := t.TempDir()
	root := filepath.Join(base, "repo")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	policy := filepath.Join(base, "policy.yml")
	if err := os.WriteFile(policy, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return root, policy
}

func TestLoadPolicy_NoPathReturnsDefaults(t *testing.T) {
	root := t.TempDir()
	cfg, err := LoadPolicy(root, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.RequireEvalCoverage || !cfg.RequireRollbackMetadata || !cfg.RequireObservability {
		t.Fatalf("missing policy should default to all-enabled, got %+v", cfg)
	}
}

func TestLoadPolicy_ExternalPolicyDisablesSubset(t *testing.T) {
	root, policy := trustedPolicyFixture(t, "require_observability: false\n")
	cfg, err := LoadPolicy(root, policy)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.RequireObservability {
		t.Fatalf("require_observability should be false, got %+v", cfg)
	}
	if !cfg.RequireEvalCoverage || !cfg.RequireRollbackMetadata {
		t.Fatalf("unspecified checks should remain enabled, got %+v", cfg)
	}
}

func TestLoadPolicy_InsideScanRootRejected(t *testing.T) {
	root := t.TempDir()
	policy := filepath.Join(root, ".ai-guards.yml")
	if err := os.WriteFile(policy, []byte("require_observability: false\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadPolicy(root, policy)
	if err == nil {
		t.Fatal("expected in-tree policy to be rejected")
	}
	if !strings.Contains(err.Error(), "outside scan root") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadPolicy_AllChecksDisabledRejected(t *testing.T) {
	root, policy := trustedPolicyFixture(t, "require_eval_coverage: false\nrequire_rollback_metadata: false\nrequire_observability: false\n")
	_, err := LoadPolicy(root, policy)
	if err == nil {
		t.Fatal("expected all-disabled policy to be rejected")
	}
	if !strings.Contains(err.Error(), "disables all checks") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadPolicy_ParentSymlinkIntoRootRejected(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "repo")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	inTreePolicy := filepath.Join(root, "policy.yml")
	if err := os.WriteFile(inTreePolicy, []byte("require_observability: false\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(base, "alias")
	if err := os.Symlink(root, alias); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	_, err := LoadPolicy(root, filepath.Join(alias, "policy.yml"))
	if err == nil {
		t.Fatal("expected policy resolving into scan root to be rejected")
	}
	if !strings.Contains(err.Error(), "resolves inside scan root") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadPolicy_SymlinkRejected(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "repo")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(base, "policy-target.yml")
	if err := os.WriteFile(target, []byte("require_observability: false\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(base, "policy.yml")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	_, err := LoadPolicy(root, link)
	if err == nil {
		t.Fatal("expected symlink policy to be rejected")
	}
	if !strings.Contains(err.Error(), "must not be a symlink") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeploymentDirectoryMatchingUsesRelativeComponents(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{path: "deploy/production.yaml", want: true},
		{path: "service/deployment/production.tf", want: true},
		{path: "deployments/production.yaml", want: false},
		{path: "production.yaml", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := hasDirectoryComponent(tt.path, "deploy", "deployment"); got != tt.want {
				t.Fatalf("hasDirectoryComponent(%q) = %t, want %t", tt.path, got, tt.want)
			}
		})
	}
}

func TestScannerRejectsSymlinkRoot(t *testing.T) {
	target := t.TempDir()
	link := filepath.Join(t.TempDir(), "repo")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	_, err := EvalCoverage(link)
	if err == nil {
		t.Fatal("expected symlink scan root to be rejected")
	}
	if !strings.Contains(err.Error(), "scan root must not be a symlink") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestScannerRejectsSymlinkedFiles(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.go")
	if err := os.WriteFile(outside, []byte("// openai call\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "model.go")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	_, err := EvalCoverage(root)
	if err == nil {
		t.Fatal("expected symlinked file to be rejected")
	}
	if !strings.Contains(err.Error(), "symlink is not allowed") {
		t.Fatalf("unexpected error: %v", err)
	}
}
