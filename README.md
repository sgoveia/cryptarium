# cryptarium

**Cryptographic discovery and CBOM generation for the post-quantum transition.**

[![CI](https://github.com/sgoveia/cryptarium/actions/workflows/ci.yml/badge.svg)](https://github.com/sgoveia/cryptarium/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/sgoveia/cryptarium.svg)](https://pkg.go.dev/github.com/sgoveia/cryptarium)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](LICENSE)

`cryptarium` finds where cryptography lives in a codebase in source, dependencies, certificates, and configuration classifies, each use by its exposure to quantum attack, and emits a standards-based **Cryptographic Bill of Materials (CBOM)** plus a prioritized migration report.

Unlike single-source scanners, `cryptarium` unifies all four evidence sources in one binary and **correlates** them, so a finding is a linked picture (a dependency, the source call that uses it, the key or certificate it produces, and the configuration that exposes it) rather than four disconnected lists.

> **Status: pre-alpha.** v0.1 is under active development. Interfaces, output schemas, and rule-pack format will change. See [DESIGN.md](DESIGN.md) for the full architecture and [the build plan](#build-plan) for where things stand.

---

## Why

You cannot migrate cryptography you cannot see. Every serious post-quantum migration framework — NIST, CISA, CNSA 2.0 — opens with the same first step: build a cryptographic inventory. In practice that step is the bottleneck. Cryptographic choices are scattered across application code, transitive dependencies, embedded certificates, TLS and SSH configuration, and infrastructure, and most of them were made implicitly years ago, defaulting to RSA or elliptic-curve primitives.

Two facts make this urgent rather than academic:

- **Harvest-now, decrypt-later.** An adversary can capture encrypted data today and decrypt it once a cryptographically-relevant quantum computer exists. Anything that must stay confidential beyond that point is already at risk health records, financial records, legal and government records, code-signing keys.
- **The replacements are standardized.** NIST finalized ML-KEM (FIPS 203), ML-DSA (FIPS 204), and SLH-DSA (FIPS 205) in 2024. The blocker is no longer which algorithm to use. It is knowing what to change.

`cryptarium` treats that blocker as what it is: a code-scanning, dependency-graph, and configuration-analysis problem that should run at engineering scale and inside CI, not a manual audit.

## Install

Pre-built binaries are not yet published. From source:

```bash
go install github.com/sgoveia/cryptarium/cmd/cryptarium@latest
```

Requires Go 1.26+.

## Usage

```bash
# Scan the current directory
cryptarium scan .

# Scan a remote repository
cryptarium scan https://github.com/OWNER/REPO

# Emit a CycloneDX CBOM
cryptarium scan . --format cbom --output cbom.json

# Emit SARIF for GitHub code scanning
cryptarium scan . --format sarif --output results.sarif

# Human-readable migration report
cryptarium scan . --format markdown --output CRYPTO-REPORT.md

# Fail CI when anything scores Critical
cryptarium scan . --fail-on critical
```

### Example output

```
cryptarium v0.1.0 — scanned 1,284 files in 2.1s

CRITICAL  services/auth/token.go:88         RSA-2048 key generation
          → broken by Shor; internet-facing; migrate to ML-DSA (FIPS 204)
          → correlated: go.mod crypto/rsa · certs/api.pem (RSA-2048 sig)

HIGH      config/nginx.conf:41              TLS_ECDHE_RSA_WITH_AES_128_GCM
          → ECDHE broken by Shor, AES-128 weakened by Grover
          → migrate to hybrid X25519+ML-KEM-768; raise to AES-256

MEDIUM    internal/store/encrypt.go:23      AES-128-GCM at rest
          → weakened by Grover; 7-year retention → HNDL exposure

  4 critical · 11 high · 23 medium · 6 low          CBOM: cbom.json
```

### GitHub Action

```yaml
- uses: OWNER/cryptarium-action@v0
  with:
    path: .
    fail-on: critical
- uses: github/codeql-action/upload-sarif@v3
  with:
    sarif_file: results.sarif
```

## What it detects

| Source | Examples | Method |
|---|---|---|
| **Source code** | Calls into `crypto/rsa`, `cryptography`, OpenSSL, BouncyCastle, Web Crypto, `java.security`; hardcoded key sizes and curves | Rule-pack pattern matching over parsed source (tree-sitter) |
| **Dependencies** | Crypto libraries from `go.mod`, `requirements.txt`, `package-lock.json`, `pom.xml`, `Cargo.toml` | Manifest/lockfile parsing against a known-library catalog |
| **Certificates & keys** | `.pem`, `.crt`, `.cer`, `.der`, `.p12`, `.jks`, SSH keys | X.509 parsing: signature algorithm, public-key algorithm, key size, curve, validity |
| **Configuration** | TLS cipher suites and versions, SSH `KexAlgorithms`, JWT `alg`, IPsec/VPN settings | Config and string pattern matching |

Every finding is classified as **Broken** (defeated by Shor's algorithm), **Weakened** (reduced margin under Grover's), or **Safe/PQC**, and mapped to a recommended migration target:

| Classical primitive | Status | Recommended target |
|---|---|---|
| RSA | Broken (Shor) | ML-KEM (FIPS 203) for key establishment; ML-DSA (FIPS 204) for signatures |
| ECDH / DH | Broken (Shor) | ML-KEM; hybrid X25519 + ML-KEM-768 during transition |
| ECDSA / EdDSA / DSA | Broken (Shor) | ML-DSA (FIPS 204); SLH-DSA (FIPS 205) where a conservative hash-based option is preferred |
| AES-128 | Weakened (Grover) | AES-256 |
| SHA-256 | Weakened (Grover) | SHA-384 / SHA-512 in high-assurance contexts |
| SHA-1, MD5, 3DES | Already broken | Deprecate immediately |

Hybrid classical-plus-PQC constructions are a first-class recommended transition state, because that is what real migrations deploy first.

## Risk-based prioritization

An inventory that flags everything as equally urgent is not actionable. Each finding is scored on four axes:

1. **Algorithm vulnerability** — broken outranks weakened outranks safe.
2. **Data longevity (HNDL exposure)** — the longer a secret must hold, the higher the harvest-now-decrypt-later risk.
3. **Exposure surface** — internet-facing vs. internal; in-transit vs. at-rest vs. code-signing.
4. **Crypto-agility** — a hardcoded primitive costs more to remediate than one behind a provider interface.

## Output formats

- **CycloneDX CBOM (1.6+)** — machine-readable inventory using `cryptographic-asset` components, consumable by any CycloneDX-aware tool.
- **SARIF** — renders in the GitHub Security tab and any SARIF viewer; can gate a build.
- **Markdown / HTML** — the human deliverable: findings grouped by service and severity with location, classification, target, and priority.

## What this is not

Credibility here depends on never overstating what static discovery can prove.

- Not a cryptographic **correctness** auditor. It reports which algorithms are used and their quantum exposure, not whether they are implemented securely (padding, IV reuse, side channels).
- Not a binary or firmware analyzer (roadmap).
- Not a runtime or network TLS scanner (roadmap). It sees what code and configuration *contain*, not what gets negotiated on the wire.
- Not a replacement for a cryptographer's judgment on migration design. It surfaces and prioritizes; humans decide.

## Build plan

| Phase | Focus | Status |
|---|---|---|
| 0 | Scaffold: CLI skeleton, `CryptoFinding` model, CI, license | ✅ |
| 1 | Certificate/key + dependency-manifest detectors; JSON + Markdown output | 🚧 |
| 2 | Rule-pack source detector (Go, Python, JS/TS, Java, C/C++); classifier; CBOM | ⬜ |
| 3 | Risk scoring; SARIF; GitHub Action | ⬜ |
| 4 | AI-assisted triage; multi-repo scanning; container images | ⬜ |

Longer roadmap: container and filesystem scanning, runtime/network discovery, binary and firmware analysis, org-wide aggregation, cloud KMS/HSM discovery, a full declarative policy engine with migration-exception tracking, and export to GitLab CI, Jenkins, SonarQube, Dependency-Track, SIEM, and GRC destinations.

## Development

The repository ships a dev container. Open it in a GitHub Codespace or in VS Code with the Dev Containers extension and you get Go 1.26, `golangci-lint`, `gotestsum`, `gofumpt`, `govulncheck`, `cyclonedx-gomod`, and the GitHub CLI, already configured.

```bash
make help      # list targets
make check     # fmt + lint + test — what CI runs
make build     # -> bin/cryptarium
make selfscan  # scan this repo with the tool itself
```

`cryptarium` must always scan its own repository cleanly. The self-scan runs on every PR.

See [DESIGN.md](DESIGN.md) for architecture and [AGENT.md](AGENT.md) for the conventions that govern both human and AI-assisted contributions.

## Contributing

The most valuable contribution is **coverage**, and coverage should be a YAML file plus a test fixture, never a Go change. Rule packs live in [`rules/`](rules/); the schema is documented in [DESIGN.md](DESIGN.md#rule-pack-schema). Adding a library, a language idiom, or a configuration format should not require touching the engine.

Every detector is validated against a `testdata/` corpus of real certificates and small sample projects. Reproducible output is the product's core promise, so a new rule without a fixture will not be merged.

## Positioning

Cryptographic discovery is an active field, not an empty one. CSNP's QRAMM toolkit (CryptoScan, CryptoDeps, TLS-Analyzer), CBOMkit, CodeQL, and Semgrep cover parts of this; IBM Quantum Safe Explorer, SandboxAQ, Keyfactor, DigiCert, and QuSecure cover the enterprise and runtime layers.

`cryptarium` aims at one layer and integrates outward from it:

| Layer | `cryptarium` position |
|---|---|
| **Code and CI discovery** | **Own it.** The open, low-friction, multi-source crypto inventory compiler for repositories and CI. |
| **Enterprise crypto posture** | **Integrate.** Export normalized CBOM/SARIF/JSON that those platforms and GRC systems ingest. |
| **Runtime and network assurance** | **Roadmap.** Correlate static intent with actually-negotiated crypto rather than claiming static analysis is complete.

> One open CLI and GitHub-native CI action that gives developers an auditable, correlated, PQC-migration-ready CBOM across all repository-resident crypto evidence — code, dependencies, certificates, and configuration — before it becomes an enterprise runtime problem.

## References

- NIST **FIPS 203** — Module-Lattice-Based Key-Encapsulation Mechanism (ML-KEM)
- NIST **FIPS 204** — Module-Lattice-Based Digital Signature Algorithm (ML-DSA)
- NIST **FIPS 205** — Stateless Hash-Based Digital Signature Algorithm (SLH-DSA)
- **CycloneDX** — Cryptography Bill of Materials (CBOM), spec 1.6+
- **NSA CNSA 2.0** — Commercial National Security Algorithm Suite and migration timeline
- **SARIF** — Static Analysis Results Interchange Format (OASIS)
- **NIST / CISA** post-quantum migration guidance — cryptographic inventory as the first step

## License

[Apache-2.0](LICENSE). The patent grant matters for cryptographic tooling.
