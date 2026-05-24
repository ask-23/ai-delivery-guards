package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ask-23/ai-delivery-guards/internal/checks"
	"github.com/ask-23/ai-delivery-guards/internal/report"
)

func main() {
	os.Exit(run(os.Args, os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr *os.File) int {
	if len(args) < 2 || args[1] != "check" {
		fmt.Fprintln(stderr, "usage: guard check --root <path>")
		return 2
	}

	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	fs.SetOutput(stderr)
	root := fs.String("root", ".", "repository root to scan")
	if err := fs.Parse(args[2:]); err != nil {
		return 2
	}

	cfg, err := checks.LoadConfig(*root)
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: load config: %v\n", err)
		return 1
	}

	enabled := checks.Enabled(cfg)
	if len(enabled) == 0 {
		fmt.Fprintln(stdout, "WARN: all checks disabled via .ai-guards.yml")
		return 0
	}

	failed := false
	for _, check := range enabled {
		findings, err := check(*root)
		if err != nil {
			fmt.Fprintf(stderr, "ERROR: %v\n", err)
			return 1
		}
		for _, f := range findings {
			fmt.Fprintf(stdout, "%s: %s: %s\n", f.Severity, f.Check, f.Message)
			if f.Severity == report.Fail {
				failed = true
			}
		}
	}
	if failed {
		return 1
	}
	return 0
}
