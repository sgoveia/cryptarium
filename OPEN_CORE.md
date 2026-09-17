# Open Core Model

This document states what in `cryptarium` is open source, what may become commercial, and the commitments that govern the boundary between them. We publish it early so that contributors and adopters know the terms before they invest time or build on the project.

> **Status:** Draft. There is no commercial offering today. Everything in this repository is Apache-2.0. This document describes how that will stay true if a commercial layer is ever built.

---

## 1. The principle

**If it helps an individual developer or a single repository find, understand, and fix quantum-vulnerable cryptography, it is open source. If it helps an organization run, track, and prove a migration program across many teams, it may be commercial.**

A developer or security engineer should get the full value of `cryptarium` on their own code without paying anything or hitting an artificial limit.

## 2. Our commitments

These commitments apply to this repository and to everything released under it.

1. **The core stays Apache-2.0, permanently.** We will not relicense the open-source core to a source-available, non-commercial, or copyleft license (for example BSL, SSPL, Elastic License, or AGPL).
2. **We will not move features out of the core.** Once a capability ships in the open-source project, it will not be removed and reintroduced as a paid feature.
3. **No crippleware.** The open-source scanner will not have artificial caps on repository size, number of findings, languages, scan frequency, or output formats.
4. **Open outputs.** Findings are emitted in open standards (CycloneDX CBOM, SARIF, JSON, Markdown). Your inventory is never locked into a proprietary format or service.
5. **Deterministic results stay open.** The detection, correlation, classification, and risk-scoring logic that produces a CBOM is open and auditable. A paid product will never be required to reproduce or verify a result.
6. **The rule-pack schema stays open.** Anyone can write, publish, and use rule packs, including outside this repository.
7. **Commercial code lives elsewhere.** Any commercial components will be developed in separate repositories. This repository will not contain license-key checks, telemetry that is on by default, or disabled "enterprise" code paths.

## 3. What is open source

Everything below is, and will remain, part of the Apache-2.0 project.

| Area | Included |
|---|---|
| **Scanning** | CLI; local path and Git URL targets; all four detectors (source code, dependencies, certificates and keys, configuration) |
| **Analysis** | Cross-source correlation; quantum-vulnerability classification; PQC migration mapping; four-axis risk scoring |
| **Outputs** | CycloneDX CBOM, SARIF, JSON findings, Markdown and HTML reports |
| **CI** | GitHub Action; fail-on-threshold policy gating; declarative policy rules evaluated per scan |
| **Extensibility** | Rule-pack schema, community rule packs, detector and reporter interfaces |
| **Roadmap items that fit the principle** | Container-image and filesystem scanning, additional languages, GitLab CI and Jenkins integrations, Dependency-Track and SonarQube export, single-repository scanning of runtime or binary targets as they are built |

## 4. What may be commercial

The following capabilities are aimed at organizations managing a migration program. If they are built as paid offerings, they will consume the open outputs above rather than replace them.

| Area | Examples |
|---|---|
| **Organization-wide visibility** | Multi-repository aggregation across an organization, central dashboard, portfolio views |
| **Program tracking** | Historical trends, migration progress over time, deadline tracking against internal or regulatory timelines |
| **Governance workflows** | Exception requests and approvals, ownership assignment, audit trails |
| **Compliance evidence** | Reports prepared for auditors and regulators (for example CNSA 2.0 and federal inventory requirements) |
| **Enterprise access** | SSO, SAML, SCIM, role-based access control |
| **Integrations for managed platforms** | GRC, SIEM, ticketing, and cryptographic-posture platform connectors |
| **Hosted services** | Hosted AI-assisted triage, managed scanning, SaaS deployment |
| **Curated content** | Premium rule packs for commercial libraries, HSM and KMS vendors, maintained on a support schedule |
| **Support** | SLAs, deployment assistance, PQC readiness assessments |

Note that the **optional AI triage layer** described in `DESIGN.md` §8 may exist in both forms: the interface and a bring-your-own-model integration can be open, while a hosted, managed version may be commercial. In either case, AI enrichment annotates findings and never originates them.

## 5. Deciding gray areas

When it is unclear where a feature belongs, we apply these questions in order:

1. **Does a single developer or single repository need it to get a correct, useful result?** If yes, it is open.
2. **Does it only become valuable once many teams, repositories, or approvers are involved?** If yes, it may be commercial.
3. **Would keeping it closed make open results harder to trust or verify?** If yes, it is open.

When in doubt, we default to open. Decisions on contested features will be discussed in a public issue before implementation.

## 6. Contributions

- All contributions to this repository are licensed under Apache-2.0, as described in Section 5 of the license.
- Contributors sign off their commits using the [Developer Certificate of Origin](https://developercertificate.org/) (`git commit -s`). We do not require a copyright-assignment CLA.
- Because contributors keep their copyright and we do not collect assignments, the project cannot be unilaterally relicensed. This is intentional and reinforces commitment 1.
- Contributions to this repository will never be moved into a closed product in a way that removes them from the open project.

## 7. Trademark

"Cryptarium" and associated logos are trademarks of the project's maintaining entity. Apache-2.0 does not grant trademark rights. You are welcome to fork, modify, and redistribute the code, but a modified distribution must not be presented as the official `cryptarium` project. Plain descriptive use (for example "compatible with cryptarium" or "rule pack for cryptarium") is fine. A full trademark policy will be published in `TRADEMARKS.md`.

## 8. Changes to this document

This document can only be changed through a public pull request, open for comment for at least **30 days** before merging. Changes may clarify or expand what is open source. Changes will not weaken the commitments in Section 2.

## 9. Questions

Open an issue with the `open-core` label if you think a feature is on the wrong side of the line, or if anything here is unclear.
