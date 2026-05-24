package main

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// repoRoot returns the absolute path of the guard module root, derived from this
// test file's location. Using runtime.Caller keeps the test independent of the
// working directory `go test` was invoked from.
func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func runGuard(t *testing.T, fixture string) (string, int) {
	t.Helper()
	root := repoRoot(t)
	cmd := exec.Command("go", "run", "./cmd/guard", "check", "--root", "testdata/"+fixture)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	exit := 0
	if ee, ok := err.(*exec.ExitError); ok {
		exit = ee.ExitCode()
	} else if err != nil {
		t.Fatalf("run guard: %v\noutput: %s", err, out)
	}
	return string(out), exit
}

func TestCLI_GoodFixtureExitsZero(t *testing.T) {
	out, exit := runGuard(t, "good")
	if exit != 0 {
		t.Fatalf("exit = %d, want 0\noutput: %s", exit, out)
	}
	if !strings.Contains(out, "PASS: eval_coverage") {
		t.Errorf("expected PASS line, got: %s", out)
	}
}

func TestCLI_BadFixtureExitsOne(t *testing.T) {
	out, exit := runGuard(t, "bad")
	if exit != 1 {
		t.Fatalf("exit = %d, want 1\noutput: %s", exit, out)
	}
	if !strings.Contains(out, "FAIL: eval_coverage") {
		t.Errorf("expected FAIL line, got: %s", out)
	}
	if !strings.Contains(out, "app/model.go") {
		t.Errorf("expected file-path citation in FAIL message, got: %s", out)
	}
}
