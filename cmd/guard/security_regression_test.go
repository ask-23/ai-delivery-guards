package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func securityRepoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func buildGuardBinary(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "guard")
	cmd := exec.Command("go", "build", "-o", bin, "./cmd/guard")
	cmd.Dir = securityRepoRoot(t)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build guard: %v\noutput: %s", err, out)
	}
	return bin
}

func runGuardBinary(t *testing.T, bin, dir string, args ...string) (string, int) {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	exit := 0
	if ee, ok := err.(*exec.ExitError); ok {
		exit = ee.ExitCode()
	} else if err != nil {
		t.Fatalf("run guard: %v\noutput: %s", err, out)
	}
	return string(out), exit
}

func TestCLI_InTreePolicyCannotDisableChecks(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".ai-guards.yml"), []byte(
		"require_eval_coverage: false\nrequire_rollback_metadata: false\nrequire_observability: false\n",
	), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "app"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "app", "model.go"), []byte("// openai call\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, exit := runGuardBinary(t, buildGuardBinary(t), root, "check", "--root", ".")
	if exit != 1 {
		t.Fatalf("exit = %d, want 1\noutput: %s", exit, out)
	}
	if !strings.Contains(out, "FAIL: eval_coverage") || !strings.Contains(out, "FAIL: observability") {
		t.Fatalf("in-tree policy disabled checks or expected failures were absent: %s", out)
	}
}

func TestCLI_RelativeRootFindsDeploymentManifest(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "deploy"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "deploy", "production.yaml"), []byte("model: v2\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, exit := runGuardBinary(t, buildGuardBinary(t), root, "check", "--root", ".")
	if exit != 1 {
		t.Fatalf("exit = %d, want 1\noutput: %s", exit, out)
	}
	if !strings.Contains(out, "FAIL: rollback_metadata") || !strings.Contains(filepath.ToSlash(out), "deploy/production.yaml") {
		t.Fatalf("relative-root manifest was not enforced: %s", out)
	}
}

func TestCLI_ExternalPolicyCanDisableSubset(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "repo")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	policy := filepath.Join(base, "policy.yml")
	if err := os.WriteFile(policy, []byte(
		"require_eval_coverage: true\nrequire_rollback_metadata: false\nrequire_observability: false\n",
	), 0o644); err != nil {
		t.Fatal(err)
	}

	out, exit := runGuardBinary(t, buildGuardBinary(t), root, "check", "--root", ".", "--policy", policy)
	if exit != 0 {
		t.Fatalf("exit = %d, want 0\noutput: %s", exit, out)
	}
	if !strings.Contains(out, "PASS: eval_coverage") {
		t.Fatalf("enabled check did not run: %s", out)
	}
	if strings.Contains(out, "rollback_metadata") || strings.Contains(out, "observability") {
		t.Fatalf("disabled checks still ran: %s", out)
	}
}
