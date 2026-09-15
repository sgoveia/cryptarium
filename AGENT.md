# AGENT.md — Working instructions for AI agents on `cryptarium`

Read this before making any change. It is the operating manual for Cursor agents, Claude Code, and any other AI assistant working in this repository.

> **Discovery note:** Cursor and most agent tooling auto-load `AGENTS.md`. Keep `AGENTS.md` as a symlink or copy of this file (`ln -sf AGENT.md AGENTS.md`), so there is exactly one source of truth.

---

## 1. What this project is

`cryptarium` is a Go CLI that discovers cryptography in a codebase — source, dependencies, certificates, configuration — classifies each use by quantum vulnerability, and emits a CycloneDX CBOM, SARIF, and a prioritized migration report.

**This is security tooling.** Its users will make migration decisions based on its output. A missed finding is a false assurance, and a wrong classification is worse than no classification. That raises the bar on everything below: correctness before features, evidence before inference, and an explicit `unknown` before a confident guess.

**Read first, in this order:**
1. `DESIGN.md` — architecture, type contracts, stage responsibilities. It is the authority on *how*.
2. `README.md` — scope, positioning, and what the tool explicitly does not do.
3. This file — conventions and rules.

## 2. Hard rules

Violating any of these means the change gets reverted, regardless of how good the rest of it is.

1. **Never invent a cryptographic claim.** Every quantum-vulnerability classification, migration target, key-size threshold, and OID must trace to FIPS 203/204/205, a NIST SP, CNSA 2.0, or an RFC — cited in a code comment. If you cannot cite it, do not assert it. Emit `ClassUnknown` and say so in the finding.

2. **Never guess to fill a gap.** Unrecognized primitive → `ClassUnknown` with evidence attached. Unparseable file → a warning plus findings-so-far. Silent omission from an inventory is the worst bug this tool can have.

3. **Never emit key material.** Private keys, secrets, and credentials are extracted for *metadata only* and discarded. Snippets are redacted before they reach any output, log, or network call. `testdata/` contains real private keys on purpose; treat them as if they were live.

4. **Never break determinism.** Same input, byte-identical output. No timestamps outside the designated metadata block, no map-iteration order in output, no counters in IDs, no dependence on goroutine scheduling. See `DESIGN.md` §9. If you touch concurrency, run the golden test twice and diff.

5. **Never let the AI layer originate a finding.** The §13 enrichment layer annotates; it never creates, deletes, or reclassifies. The CBOM must be identical with it on and off.

6. **Never widen scope past the current phase** without saying so. Check `README.md` for the active phase. Runtime scanning, binary analysis, and container support are roadmap items — implementing one early destabilizes the pipeline everything else depends on.

7. **No new third-party dependency without justification.** State what it does, why the standard library is insufficient, its license, and its maintenance status, in the PR description. `crypto/x509` already covers most certificate work. A crypto-inventory tool with a sprawling dependency graph is self-undermining.

8. **Never commit secrets.** No API keys, no tokens, no `.env`. `CURSOR_API_KEY` and `ANTHROPIC_API_KEY` come from Codespaces secrets and stay in the environment.

## 3. Workflow expectations

**Plan before you code.** For anything beyond a localized fix, produce a plan and wait for approval. State which pipeline stage you are touching, which types change, what tests you will add, and what you are deliberately not doing.

**One stage at a time.** The pipeline stages have narrow contracts on purpose. A change that touches the detector, the classifier, and the reporter at once is almost always three changes wearing a trenchcoat. Split it.

**Tests in the same change as the code.** Not after. A detector rule without a positive *and* a negative fixture will not be merged; the negative fixture is the one that matters, because it encodes what the rule must not match.

**Run `make check` before declaring done.** That is `fmt`, `lint`, and `test` — the same thing CI runs. "It compiles" is not done.

**Keep `DESIGN.md` true.** If a change contradicts the design doc, update the doc in the same PR. A stale design doc poisons every future agent session, including yours.

**Ask instead of resolving open questions.** `DESIGN.md` §17 lists decisions deliberately left open. Do not quietly pick one. Surface it.

## 4. Go conventions

- **Go 1.26.** Standard library first, always.
- **Formatting:** `gofumpt` + `goimports`. `make fmt` before committing.
- **Linting:** `golangci-lint run` must be clean. Do not add `//nolint` without a comment explaining why — and "the linter is wrong" needs to say *how*.
- **Errors:** wrap with context (`fmt.Errorf("parse %s: %w", path, err)`). Never `panic` in library code. Never discard an error with `_` outside tests.
- **Context:** every potentially-slow call takes `ctx context.Context` as its first parameter and honors cancellation. A scan must be interruptible.
- **Interfaces:** define at the consumer, keep them small, accept interfaces and return structs.
- **Naming:** exported identifiers have doc comments starting with the identifier name. Crypto primitive names are canonical and uppercase (`RSA`, `ECDSA`, `AES`, `SHA-256`) — never free-text.
- **Concurrency:** stateless detectors, bounded worker pools, results sorted at the join point. `go test -race` is not optional.
- **No global mutable state** except the init-time detector registry.
- **Table-driven tests** with named cases. `gotestsum --format testname` is the local runner.

## 5. Adding coverage — do this the declarative way

The single most common task in this repo is "detect crypto in X". Before writing Go:

**Can it be a rule pack?** Adding a library, a language idiom, a config format pattern, or a cipher-suite string should be a YAML file in `rules/` plus fixtures in `testdata/rules/<rule-id>/`. That is the design intent (`DESIGN.md` §5) and the community contribution surface. If you find yourself editing Go to add coverage, stop and ask whether the *rule schema* should be extended instead — that change helps every future rule, while a hardcoded detector helps one.

**Checklist for a new rule:**

- [ ] Unique, stable `id` (renaming one breaks users' SARIF baselines)
- [ ] Canonical `primitive` name that exists in `internal/classify/canonical.go`
- [ ] Honest `confidence` — `high` only when the match alone establishes primitive *and* parameters
- [ ] Positive fixture that must match
- [ ] Negative fixture that must not (comment, test file, similar-looking non-crypto call)
- [ ] `references` link to authoritative documentation
- [ ] `make lint-rules` passes and the golden test is regenerated deliberately

## 6. Commits and PRs

Conventional Commits:

```
feat(detector/certs): parse PKCS#12 bundles
fix(score): default longevity to 50 when no signal exists
docs(design): record tree-sitter CGO decision
test(rules): negative fixture for go.crypto.rsa.generatekey in comments
chore(deps): bump cyclonedx-gomod
```

Scopes match package names: `collector`, `detector/source`, `detector/deps`, `detector/certs`, `detector/config`, `correlate`, `classify`, `score`, `policy`, `rules`, `report`, `cmd`.

PR descriptions state: what changed, which pipeline stage, whether any type in `internal/model` changed (breaking), what tests were added, and what was deliberately left out.

**Do not push or merge autonomously.** Commit locally, leave the push to a human. `git push`, `git reset --hard`, and `gh pr merge` are denied in the agent permission config for this reason.

## 7. Commands

```bash
make help       # list targets
make build      # -> bin/cryptarium
make test       # gotestsum with -race and coverage
make lint       # golangci-lint
make fmt        # gofumpt + goimports
make check      # fmt + lint + test — run this before saying "done"
make selfscan   # scan this repo with the tool itself
make golden     # regenerate golden files — review the diff, never blind-accept
make vuln       # govulncheck
```

Environment: GitHub Codespaces with the dev container in `.devcontainer/`. Go, linters, and `cursor-agent` are pre-installed. Never install tooling into `~` ad hoc — it disappears on rebuild. Add it to `.devcontainer/post-create.sh` instead.

## 8. Where things live

| I want to... | Go here |
|---|---|
| Add a detected library or language pattern | `rules/` (YAML) + `testdata/rules/` |
| Change what a detector extracts | `internal/detector/<kind>/` |
| Change how a primitive is classified | `internal/classify/` — **with a citation** |
| Change prioritization | `internal/score/` |
| Change output format | `internal/report/<format>/` |
| Add a CLI flag | `cmd/cryptarium/` — parsing only, no logic |
| Change a core type | `internal/model/` — breaking; flag it in the PR |
| Change how findings are linked | `internal/correlate/` |

## 9. Anti-patterns seen in this problem space

Specific failure modes worth naming, because they are easy to walk into:

- **Regex-only source detection.** It matches comments, strings, and test fixtures, and the resulting false-positive rate makes teams turn the tool off. Parse the AST; use regex only as an explicit fallback with lowered confidence.
- **Flagging everything as critical.** An inventory with no prioritization is a spreadsheet nobody acts on. That is precisely the status quo this tool exists to replace.
- **Treating a dependency as proof of use.** A crypto library in `go.mod` is `medium` confidence at most. Correlation with a source finding is what earns `high`.
- **Silently skipping unparseable files.** Report them. A user needs to know the inventory has a hole.
- **Claiming completeness.** Static analysis cannot see runtime algorithm selection, dynamically loaded providers, or what actually gets negotiated on the wire. The output must never imply otherwise.
- **Over-abstracting early.** Four detectors, one interface. Do not build a plugin framework for three implementations.
- **Letting the AI layer leak into the core.** If enrichment can change a classification, the CBOM stops being reproducible and the whole value proposition goes with it.

## 10. Tone of the output itself

User-facing text — CLI messages, Markdown reports, SARIF descriptions — is written for an engineer who has to act on it. Plain, specific, no alarmism. State the finding, the evidence, the reason, the recommended target, and the confidence. "RSA-2048 key generation at token.go:88; broken by Shor's algorithm; migrate to ML-DSA-65 (FIPS 204)" is right. "⚠️ CRITICAL QUANTUM THREAT DETECTED" is not — and it is the fastest way to lose the audience this tool needs.
