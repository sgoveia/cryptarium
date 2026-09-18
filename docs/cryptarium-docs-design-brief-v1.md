# cryptarium — Mintlify Docs Design Brief (v1)

**Target Aesthetic:** Institutional trust, evidence-first, CI-native security tooling
**Site:** Mintlify-hosted developer documentation for the `cryptarium` CLI
**Purpose:** Instructions for generating Mintlify `mint.json` (or `docs.json`) configuration and `.mdx` content files for Cryptarium documentation. This brief is read by agents when scoping or implementing any work inside `docs/`.

> **Provenance:** adapted from `wavelet-docs-design-brief-v1.md` (structure, transferable Mintlify patterns, acceptance discipline). Product surface, brand, palette, IA, and tone are rewritten for Cryptarium — a Go CLI security tool, not a hosted API. The Wavelet brief remains in-tree as provenance only; **this document is the docs source of truth.**

---

## 1. Brand Identity & Visual Language

The design must evoke **institutional trust, technical sophistication, and evidence-based inventory**. The layout should guide a security engineer from "what is this" to "I have a CBOM and a prioritized report" without friction.

- **Vibe:** B2B security infrastructure, audit-ready, modern, CI-native. The reader is a security engineer, AppSec lead, platform/crypto migration owner, or contributor — never a marketing visitor.
- **Core Metaphor:** **Cryptographic inventory for the post-quantum transition.** Evidence, classification, correlation, and migration targets — not fluid trading, waves, or XRPL. The name suggests a place where crypto is collected and examined; the UI should feel like a precise inventory tool, not a threat dashboard.
- **What the docs are not:** they are not a sales site. There is no "Schedule a demo," no "Talk to sales," no gated content. Every page either teaches the engineer something or lets them install and scan in one click.

---

## 2. Color Palette (Hex Codes)

Configure `mint.json` using these colors derived from the Cryptarium logo (`cryptarium_logo.svg`). Do not import Ripple / Wavelet blues.

| Role | Hex | Usage |
|---|---|---|
| **Primary Action (Solid)** | `#005DFC` (Electric Blue) | Standard buttons, active tabs, primary links, `colors.primary` in `mint.json`, in-doc `<Card>` icons by default. |
| **Light Accent** | `#3D7FFF` | `colors.light`, hero highlights, anchors gradient start. Use sparingly. |
| **Dark / Navy** | `#031F45` | `colors.dark`, dark-mode body, wordmark fill on light backgrounds, anchors gradient end. |
| **Background (Light)** | `#FFFFFF` (Pure White) + `#F5F8FC` (Navy Tint) | Primary background pure white; alternating `<CardGroup>` rows and tip blocks use Navy Tint. |
| **Background (Dark)** | `#031F45` (Navy) | Dark mode body — deep navy, not teal. Code blocks sit on top without harsh contrast. |
| **Text (Primary)** | `#2A3340` (Deep Slate) | Body copy on light backgrounds. WCAG AA against white. |
| **Accents — Success** | `#0D9F6E` | Success states, "scan complete" examples, check icons in capability tables. |
| **Accents — Info** | `#005DFC` | Informational callouts (`<Info>`), determinism markers, classification badges for Safe/PQC. |

> **Logo fidelity:** the mark uses Navy `#031F45` for the geometric C and wordmark, and Electric Blue `#005DFC` for the quantum/atom glyph. Never recolor the logo files. Docs chrome uses the same two anchors so the site and the mark read as one brand.

---

## 3. Typography

Cryptarium docs present dense technical material — CLI flags, finding tables, CBOM JSON, rule-pack YAML, SARIF property docs. Type must be highly legible at small sizes and authoritative at large sizes.

- **Primary Typeface:** `Inter` for both `font.headings` and `font.body` in `mint.json`. (The logo SVG may use Avenir Next / Montserrat for the wordmark; that stays in the mark only.)
- **Headings (`h1`, `h2`, `h3`):** bold with tight tracking. A CLI reference page should scan in 5 seconds (Synopsis, Flags, Exit codes, Examples).
- **Body text:** generous line-height (`1.6+`). Classification rationale and detector caveats are read line-by-line; cramped text destroys comprehension.
- **Monospace (code):** `JetBrains Mono`, `Fira Code`, or `IBM Plex Mono`. Ligatures on. Used for: CLI invocations, file paths, rule IDs (`go.crypto.rsa.generatekey`), flags (`--fail-on`), JSON/YAML, and primitive names (`RSA`, `ML-DSA-65`).

---

## 4. Logo & Iconography

### 4.1 Logo

Source assets live in `docs/`:

- `cryptarium_logo.svg` — full mark (geometric C + atom + wordmark)
- `logo.png` — raster equivalent

Use as-is, never recolored, never stretched. Mintlify variants (produced in site setup):

- `logo/light.svg` — colored mark + navy wordmark, for light backgrounds (`logo.light`)
- `logo/dark.svg` — colored mark + white wordmark, for dark backgrounds (`logo.dark`)
- `favicon.svg` — geometric C + atom glyph only, square-cropped

### 4.2 Iconography

Use Mintlify's FontAwesome integration. Icons should be **minimal, line-style, and rendered in the primary electric blue** (`#005DFC`) on light mode. Avoid duotone, avoid filled glyphs, avoid emoji.

Canonical icon mapping — agents should reuse these when introducing the same concept twice:

| Concept | FontAwesome icon |
|---|---|
| Quickstart / first scan | `rocket` |
| Install | `download` |
| Scan command | `magnifying-glass` |
| Source detector | `code` |
| Dependency detector | `cubes` |
| Certificates & keys | `certificate` |
| Configuration | `sliders` |
| Correlation | `link` |
| Quantum classification | `shield-halved` |
| Confidence / unknown | `circle-question` |
| Determinism | `clock-rotate-left` |
| Risk scoring | `gauge` |
| CBOM output | `file-code` |
| SARIF output | `bug` |
| Markdown / HTML report | `file-lines` |
| JSON output | `brackets-curly` |
| CLI reference | `terminal` |
| GitHub Action / CI | `github` |
| Rule packs | `book` |
| Adding a rule | `plus` |
| Policy / fail-on | `ban` |
| Limitations / roadmap | `map` |
| Errors / warnings | `circle-exclamation` |

---

## 5. UI Elements & Layout Strategy (Mintlify Specifics)

Match a spacious, card-led layout using Mintlify's built-in components — adapted for a CLI product, not an API catalog.

### 5.1 `<CardGroup>` & `<Card>`

Use extensively. The landing page's first interactive element should be a **four-card `<CardGroup cols={4}>`** for the four evidence sources:

```mdx
<CardGroup cols={4}>
  <Card title="Source" icon="code" href="/concepts/evidence-sources">
    Rule-pack detection over parsed source (tree-sitter).
  </Card>
  <Card title="Dependencies" icon="cubes" href="/concepts/evidence-sources">
    Manifest and lockfile crypto-library inventory.
  </Card>
  <Card title="Certificates" icon="certificate" href="/concepts/evidence-sources">
    X.509 and key metadata — algorithm, size, curve, validity.
  </Card>
  <Card title="Configuration" icon="sliders" href="/concepts/evidence-sources">
    TLS, SSH, JWT, and related crypto settings.
  </Card>
</CardGroup>
```

Secondary `<CardGroup>` blocks for outputs (CBOM · SARIF · Report · JSON) and for getting started (Install · Quickstart · CI).

### 5.2 Whitespace

Every MDX section breathes. Tables (findings, flags, classification) get vertical padding so reference data scans cleanly.

### 5.3 Code Blocks

Use Mintlify's native `<CodeGroup>` wherever a runnable example exists. **Canonical tab order for Cryptarium:**

1. **bash** — install and `cryptarium scan` (always first)
2. **yaml** — GitHub Action / rule-pack snippets when relevant
3. **go** — only when embedding or library use applies (rare in v0.1)

Do **not** default to cURL / TypeScript / Python SDK tabs — this is not an HTTP API product.

```mdx
<CodeGroup>

```bash
# Scan the current directory
cryptarium scan .

# Emit a CycloneDX CBOM
cryptarium scan . --format cbom --output cbom.json
```

```yaml
- uses: sgoveia/cryptarium@v0.2.0
  with:
    path: .
    fail-on: critical
    upload-sarif: true
```

</CodeGroup>
```

Syntax highlighting theme: deep navy dark-mode palette (`#031F45`), never stark black. Recommended: `night-owl` or `material-palenight` tuned toward Electric Blue accents.

### 5.4 `<AccordionGroup>`

Used for: flag reference (one accordion per flag group), FAQ on limitations, and per-detector caveats.

### 5.5 `<Tabs>` (where `<CodeGroup>` doesn't fit)

Use `<Tabs>` for genuinely different content paths (e.g., install via `go install` vs GitHub Releases vs building from source). Do **not** use `<Tabs>` where `<CodeGroup>` is more appropriate — `<CodeGroup>` is the same example in different formats; `<Tabs>` is different procedures.

### 5.6 Hero Sections

The landing page hero is **center-aligned, high-contrast typography** that communicates value in one breath:

> **Cryptographic discovery and CBOM generation for the post-quantum transition.**
> One CLI. Four evidence sources. Correlated findings. Standards-based CBOM, SARIF, and a prioritized migration report.

Below the hero: the four-card evidence-source `<CardGroup>`. Below that: a one-paragraph TL;DR with a single primary CTA (`Install →` linking to `/install`). No secondary sales CTA.

### 5.7 Custom Components (acceptable additions)

When Mintlify's built-ins don't cover the need, build small React/MDX components inside `docs/components/` and use them sparingly:

- **`<ClassificationBadge/>`** — Broken / Weakened / Safe / Unknown, styled per §2 (no alarmist red chrome for Broken; use restrained severity colors).
- **`<FindingExample/>`** — standardized finding line: location, primitive, class, recommendation, confidence.
- **`<OutputFormatTabs/>`** — CBOM / SARIF / Markdown / JSON when explaining the same scan with different `--format` values.

No pricing calculators. No API key signup widgets.

---

## 6. Documentation Structure (Navigation Tree)

This is the canonical `navigation` shape for `mint.json`. Agents building or refactoring pages under `docs/` should reuse this tree; new pages get slotted into the existing group rather than introducing parallel hierarchies.

```
Home  (/)
├── Quickstart                          (/quickstart)
├── Install                             (/install)
└── What Cryptarium is not              (/what-it-is-not)

Concepts
├── Four evidence sources               (/concepts/evidence-sources)
├── Correlation                         (/concepts/correlation)
├── Quantum classification              (/concepts/classification)
├── Confidence & unknown                (/concepts/confidence)
├── Determinism                         (/concepts/determinism)
└── Risk scoring                        (/concepts/scoring)

Outputs
├── Overview                            (/outputs/overview)
├── CBOM (CycloneDX)                    (/outputs/cbom)
├── SARIF                               (/outputs/sarif)
├── Markdown / HTML report              (/outputs/report)
└── JSON                                (/outputs/json)

CLI reference
├── Overview                            (/cli/overview)
├── cryptarium scan                     (/cli/scan)
├── Flags & exit codes                  (/cli/flags)
└── GitHub Action                       (/cli/github-action)

Detectors & rules
├── Detectors overview                  (/detectors/overview)
├── Rule packs                          (/detectors/rule-packs)
├── Adding a rule                       (/detectors/adding-a-rule)
└── Fixtures & validation               (/detectors/fixtures)

Guides
├── First scan in 5 minutes             (/guides/first-scan)
├── CI with --fail-on                   (/guides/ci-fail-on)
├── Uploading SARIF                     (/guides/sarif-upload)
└── Reading the migration report        (/guides/migration-report)

Reference
├── Canonical primitives                (/reference/primitives)
├── Limitations & roadmap               (/reference/limitations)
└── Changelog                           (/reference/changelog)
```

Top-bar primary nav surfaces three groups: **Docs · CLI · Outputs**. The "Install" button sits in the top-right and links to `/install`. Search is required and lives in the header. GitHub is an anchor linking to `https://github.com/sgoveia/cryptarium`.

Design briefs (`cryptarium-docs-design-brief-v1.md`, `wavelet-docs-design-brief-v1.md`) stay in the repo but are **excluded from navigation and from the Mintlify build** via `.mintignore` (so MDX examples inside the briefs do not fail `mint validate`).

---

## 7. Content & Tone

### 7.1 Voice

- **Authoritative, engineer-first, security inventory.** No exclamation points. No marketing fluff. No emoji in body copy. No alarmist quantum-threat language.
- **Show, don't sell.** A finding line beats any adjective:

  > RSA-2048 key generation at `token.go:88`; broken by Shor's algorithm; migrate to ML-DSA-65 (FIPS 204).

- **Honest about scope.** Static analysis cannot see runtime algorithm selection, dynamically loaded providers, or what gets negotiated on the wire. Say so on the relevant page. Hidden caveats destroy trust.
- **Never invent cryptographic claims.** Classifications, migration targets, key-size thresholds, and OIDs must trace to FIPS 203/204/205, a NIST SP, CNSA 2.0, or an RFC — or use `unknown`. Docs must match product doctrine: evidence before inference.

### 7.2 Structural conventions

- **Every concept page** opens with a one-sentence definition, then a "Why this matters" paragraph (≤80 words), then the technical content.
- **Every CLI / output page** follows this fixed skeleton: purpose → when to use → worked example (`<CodeGroup>`) → fields / flags → caveats → Related links.
- **Every guide** is a working, copy-paste-runnable narrative. The reader who copies every block in order ends with a completed scan or CI gate. No partial snippets.
- **Detector / rule pages** state confidence honestly and include positive and negative fixture expectations.

### 7.3 What to break down

Bite-sized sections for topics that otherwise overwhelm:

- **Four evidence sources:** what each detector sees, what it cannot see, confidence ceilings (e.g., dependency-only = medium at most until correlated).
- **Correlation:** how a dep + call site + cert + config become one finding picture.
- **Quantum classes:** Broken (Shor) vs Weakened (Grover) vs Safe/PQC vs Unknown — with citation pointers, not invented thresholds.
- **Scoring axes:** algorithm vulnerability, data longevity (HNDL), exposure surface, crypto-agility.
- **Outputs:** when to choose CBOM vs SARIF vs Markdown vs JSON.
- **Rule packs:** schema fields, stable `id`, canonical `primitive`, fixtures, `make lint-rules`.

---

## 8. `docs.json` Setup Directives

Mintlify now uses **`docs.json`** (not the deprecated `mint.json`). Generate configuration along these lines (illustrative; tune as Mintlify versions evolve):

```json
{
  "$schema": "https://mintlify.com/docs.json",
  "theme": "mint",
  "name": "cryptarium",
  "colors": {
    "primary": "#005DFC",
    "light":   "#3D7FFF",
    "dark":    "#031F45"
  },
  "logo": {
    "light": "/logo/light.svg",
    "dark":  "/logo/dark.svg"
  },
  "favicon": "/favicon.svg",
  "fonts": { "family": "Inter" },
  "navbar": {
    "primary": { "type": "button", "label": "Install", "href": "/install" }
  },
  "navigation": {
    "anchors": [
      { "anchor": "Docs", "icon": "book-open", "groups": [] },
      { "anchor": "CLI", "icon": "terminal", "groups": [] },
      { "anchor": "Outputs", "icon": "file-code", "groups": [] },
      { "anchor": "GitHub", "icon": "github", "href": "https://github.com/sgoveia/cryptarium" }
    ]
  },
  "footer": {
    "socials": { "github": "https://github.com/sgoveia/cryptarium" }
  }
}
```

Required behaviors:

- `colors.primary` **must** be `#005DFC`. Do not override without explicit founder approval.
- **No OpenAPI mandate.** Cryptarium is a CLI. Sources of truth are: CLI flags (`DESIGN.md` / `cmd/cryptarium`), rule packs in `rules/`, and classification tables in `internal/classify` — documented in MDX, not auto-generated from an HTTP spec.
- Dark mode must be available and visually first-class. Many readers run dark IDEs and review SARIF in dark GitHub UI.
- Search required in the header (Mintlify-hosted search; local preview may prompt `mint login`).
- Keep design briefs and raw logo sources in `.mintignore`.

---

## 9. Acceptance Criteria (for any PR that touches `docs/`)

A PR is mergeable when:

- [ ] `docs.json` validates and renders locally via `mint validate` / `mint dev`.
- [ ] `colors.primary` is `#005DFC`; no hard-coded color overrides outside the palette in §2.
- [ ] Concept / CLI / output / guide pages follow the §7.2 skeletons.
- [ ] Executable examples use the canonical `<CodeGroup>` order (bash → yaml → go when applicable).
- [ ] Iconography uses the §4.2 mapping; new concepts either reuse a mapping or extend this table in the same PR.
- [ ] No "Talk to sales" / "Schedule a demo" / "Contact us for pricing" copy anywhere.
- [ ] No alarmist quantum-threat marketing copy; findings are plain and specific.
- [ ] Dark mode renders cleanly (no contrast failures against navy `#031F45`).
- [ ] Mintlify search returns the new page for its primary keyword.
- [ ] Crypto claims in docs match README/DESIGN or are marked unknown — no invented FIPS mappings.
- [ ] Design brief files are not listed in public navigation.

---

*This document is the source of truth for Cryptarium's public documentation design. Edit when the brand evolves, the CLI surface materially changes, or Mintlify ships components that supersede the patterns here. Trivial copy edits inside individual MDX files do not require an update to this brief.*
