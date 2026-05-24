# ai-delivery-guards

A small Go CLI for mechanical CI checks around AI-touched changes.

The tool is intentionally simple. It scans a repository for LLM call-site markers and checks whether the surrounding delivery evidence exists: eval coverage, rollback metadata, and observability.

This is not an AI safety framework. It is a delivery guardrail demonstration.

## Checks

| Check               | Mechanism                                                                                                                                                                                                                     | Fails when                                                    |
| ------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------- |
| `eval_coverage`     | Scans for `openai`/`anthropic`/`llm.call`/`model.invoke` markers, then requires a directory whose exact basename is `eval`, `evals`, `evaluation`, or `evaluations`                                                           | An LLM call site exists but no eval directory does            |
| `rollback_metadata` | Locates deployment manifests (basename `manifest*`, OR `.yml`/`.yaml`/`.json`/`.tf`/`.toml`/`.hcl`/`.template` under `deploy/` or `deployment/`), then requires `rollback`/`fallback_model`/`previous_model` inside that file | A manifest exists but contains no rollback declaration        |
| `observability`     | For each LLM call-site file, requires `trace_id`/`span`/`token_usage`/`model_latency`/`llm_latency` **in the same file**                                                                                                      | A call site lacks an instrumentation marker in its own source |

Findings include the offending file path and the matched marker so CI failures are actionable without re-running locally.

## Run

```bash
go run ./cmd/guard check --root .
```

Exit codes:

- `0` — no failures
- `1` — one or more checks failed
- `2` — invalid command usage

## Configuration

Optional `.ai-guards.yml` at the scan root toggles individual checks:

```yaml
require_eval_coverage: true
require_rollback_metadata: true
require_observability: true
```

Missing keys default to `true`. A missing file leaves every check enabled. Inline `# comments` after values are supported. An invalid boolean value (e.g. a typo) errors out rather than silently disabling a guard.

## Default exclusions

The walker skips: `.git/`, `vendor/`, `node_modules/`, `internal/checks/`, and `testdata/`. This prevents the guard from self-satisfying its own checks when scanning its own repository.

## Example CI

See `examples/github-action.yml`.

## Intentional limits

- Marker-level scanning (substring match), not AST analysis. A determined developer can satisfy the checks superficially. The point is to make accidental negligence visible in CI.
- No central policy server.
- No vendor-specific AI SDK integration.
- No SaaS dependency.

The first version proves the operating model, not a full governance platform.
