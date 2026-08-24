# ai-delivery-guards

A small Go CLI for mechanical CI checks around AI-touched changes.

The tool is intentionally simple. It scans a repository for LLM call-site markers and checks whether the surrounding delivery evidence exists: eval coverage, rollback metadata, and observability.

This is not an AI safety framework. It is a delivery guardrail demonstration.

## Checks

| Check               | Mechanism                                                                                                                                                                                                                     | Fails when                                                    |
| ------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------- |
| `eval_coverage`     | Scans for `openai`/`anthropic`/`llm.call`/`model.invoke` markers, then requires a directory whose exact basename is `eval`, `evals`, `evaluation`, or `evaluations`                                                           | An LLM call site exists but no eval directory does            |
| `rollback_metadata` | Locates deployment manifests (basename `manifest*`, OR `.yml`/`.yaml`/`.json`/`.tf`/`.toml`/`.hcl`/`.template` under a `deploy/` or `deployment/` path component), then requires `rollback`/`fallback_model`/`previous_model` inside that file | A manifest exists but contains no rollback declaration        |
| `observability`     | For each LLM call-site file, requires `trace_id`/`span`/`token_usage`/`model_latency`/`llm_latency` **in the same file**                                                                                                      | A call site lacks an instrumentation marker in its own source |

Findings include the offending file path and the matched marker so CI failures are actionable without re-running locally.

## Run

All checks are enabled by default:

```bash
go run ./cmd/guard check --root .
```

A trusted policy file can disable a subset:

```bash
go run ./cmd/guard check \
  --root . \
  --policy "$HOME/.config/ai-delivery-guards/policy.yml"
```

Exit codes:

- `0` — no failures
- `1` — one or more checks failed, the policy was unsafe, or the scan could not be completed
- `2` — invalid command usage

## Policy and trust boundary

The scanned repository is treated as untrusted input. A repository-local `.ai-guards.yml` is not loaded. The only configuration that may disable checks is an explicitly supplied `--policy` file, and that file must:

- be a regular file rather than a symlink;
- exist outside the scan root, both lexically and after path resolution;
- leave at least one check enabled.

Policy syntax is intentionally minimal:

```yaml
require_eval_coverage: true
require_rollback_metadata: true
require_observability: false
```

Missing keys default to `true`. Inline `# comments` after values are supported. An invalid boolean errors rather than silently changing enforcement. See `examples/policy.yml`.

For adversarial pull requests, the guard executable and workflow must also come from a trusted, pinned revision. Keeping the policy outside the checkout prevents the subject repository from weakening configuration; it does not make a guard binary built from attacker-controlled source trustworthy.

## Filesystem behavior

The scanner fails closed on filesystem ambiguity:

- symbolic links in the scan root are rejected;
- a symbolic-link scan root is rejected;
- unreadable files produce an error rather than being skipped.

The walker skips `.git/`, `vendor/`, `node_modules/`, `internal/checks/`, and `testdata/`. This prevents the guard from self-satisfying its own checks when scanning its own repository.

## Example CI

See `examples/github-action.yml`.

## Intentional limits

- Marker-level scanning (substring match), not AST analysis. A determined developer can satisfy the semantic checks superficially. The point is to make accidental negligence visible in CI.
- No central policy server.
- No vendor-specific AI SDK integration.
- No SaaS dependency.

The first version proves the operating model, not a full governance platform.
