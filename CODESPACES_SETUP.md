# Cryptarium — GitHub Codespaces Setup Guide

A step-by-step path from an empty GitHub account to a working `cryptarium` development environment in Codespaces, driven entirely from a laptop browser (or VS Code Desktop attached to the Codespace), with the Cursor CLI agent running inside the container.

**Assumptions**

- All code lives in one GitHub repo: `cryptarium`.
- The laptop is a thin client. No Go toolchain, no Cursor IDE, no build tooling installed locally. The only local requirements are a browser and (optionally) `gh` and VS Code Desktop.
- Development is done with the Cursor CLI (`cursor-agent`) running **inside** the Codespace, in the integrated terminal.

**Time to first green build:** about 20 minutes, most of it waiting on the first container build.

---

## Step 0 — Accounts and one-time prerequisites

| Item | Why | Where |
|---|---|---|
| GitHub account with Codespaces enabled | The dev environment | Free tier includes monthly core-hours; a paid plan is worth it for a 4-core machine |
| Cursor account with CLI access | `cursor-agent` requires a logged-in Cursor plan | https://cursor.com |
| Cursor API key | Headless auth inside the Codespace | Cursor dashboard → Integrations / API keys → create key (starts `crsr_...`) |
| (Optional) `gh` CLI on the laptop | Create the repo and SSH into Codespaces from the terminal | `brew install gh` / `winget install GitHub.cli` |
| (Optional) VS Code Desktop + "GitHub Codespaces" extension | Nicer editor than the browser tab; same remote container | https://code.visualstudio.com |

Generate the Cursor API key now and keep it on the clipboard. Step 3 consumes it.

---

## Step 1 — Create the `cryptarium` repository

**Option A — from the laptop with `gh`:**

```bash
gh auth login
gh repo create cryptarium \
  --public \
  --license apache-2.0 \
  --gitignore Go \
  --description "Cryptographic discovery and CBOM generation for the post-quantum transition." \
  --clone
cd cryptarium
```

**Option B — from the browser:** github.com → New repository → name `cryptarium`, Public, add README, add `.gitignore: Go`, add license `Apache License 2.0`.

Apache-2.0 is the decision recorded in the design brief (§14), for the patent grant. Set it at creation so `LICENSE` is in place from the first commit.

---

## Step 2 — Seed the repo with the config and docs

Add these files, either locally (Option A above) or via the browser's "Add file → Create new file":

```
cryptarium/
├── .devcontainer/
│   ├── devcontainer.json
│   ├── post-create.sh
│   └── post-start.sh
├── .github/workflows/ci.yml
├── AGENT.md
├── AGENTS.md          -> copy of AGENT.md (Cursor auto-discovers this name)
├── DESIGN.md
└── README.md
```

The contents of everything under `.devcontainer/` are in the companion document `DEVCONTAINER_CONFIG.md`. `AGENT.md`, `DESIGN.md`, and `README.md` are the other three artifacts in this set.

Commit and push:

```bash
git add .
git commit -m "chore: devcontainer, agent instructions, design brief"
git push
```

> **Why this order matters.** A Codespace is created *from a commit*. If `.devcontainer/` is not pushed before you launch, you get the default universal image and none of the toolchain. Push first, launch second.

---

## Step 3 — Configure Codespaces secrets

The Codespace is headless, so `cursor-agent` cannot open a browser to complete an interactive login. Supply the API key as an encrypted secret instead. It is injected as an environment variable into every Codespace for this repo, on every start, with no key ever committed.

1. Go to **https://github.com/settings/codespaces**.
2. Under **Codespaces secrets**, click **New secret**.
3. Name: `CURSOR_API_KEY`. Value: the `crsr_...` key from Step 0.
4. Repository access: select `cryptarium` (or "All repositories" if you prefer).

Add these while you are there if you want them:

| Secret | Purpose |
|---|---|
| `CURSOR_API_KEY` | Cursor CLI headless auth — **required** |
| `ANTHROPIC_API_KEY` | For the optional AI triage layer in §8 of the design brief, once you build it |
| `GH_TOKEN` | Only if you need a PAT with wider scope than the built-in Codespaces token |

A secret change takes effect on the **next** Codespace start. If a Codespace is already running, stop and restart it.

---

## Step 4 — (Optional but recommended) Set up a personal dotfiles repo

`.devcontainer/` is project configuration: everyone who opens `cryptarium` gets it. Dotfiles are *personal* configuration: your shell, aliases, git identity, and editor preferences, applied automatically to **every** Codespace you create, in any repo.

1. Create a public repo named `dotfiles` on your account.
2. Add the files listed in `DEVCONTAINER_CONFIG.md` (section "Dotfiles repository").
3. Go to **https://github.com/settings/codespaces** → **Dotfiles** → check **Automatically install dotfiles**.

Codespaces clones the repo into `~/dotfiles` and runs `install.sh` after the container is created. Keep it fast and idempotent; a slow or failing dotfiles script silently degrades every Codespace you launch.

**Rule of thumb for what goes where:**

| Goes in `.devcontainer/` | Goes in `dotfiles` |
|---|---|
| Go toolchain, linters, `cursor-agent` | Shell prompt, aliases, `.zshrc` |
| Anything a contributor needs to build the project | `git` name/email, `git` aliases |
| Anything CI should also have | Personal `cursor-agent` defaults |

---

## Step 5 — Launch the Codespace

From the repo page: **Code ▾ → Codespaces → Create codespace on main**.

Or from the laptop terminal:

```bash
gh codespace create --repo <your-user>/cryptarium --machine standardLinux32gb
gh codespace code   # opens in VS Code Desktop
# or
gh codespace ssh    # plain terminal, no editor at all
```

**Machine size.** 2-core works. 4-core (`standardLinux32gb`) is the right default here: tree-sitter builds with CGO, and the fleet-scale scanning work in this project is concurrency-heavy, so you want cores to test against. You can change machine type later from the Codespaces page, which rebuilds onto a new VM.

The first build takes 3–6 minutes: it pulls the base image, installs features, then runs `post-create.sh`. Watch the log via **"Building codespace"** → **View log**. Read that log the first time; almost every setup problem is visible there in plain text.

---

## Step 6 — Verify the environment

In the Codespace terminal:

```bash
go version                 # go1.26.x
golangci-lint --version
gotestsum --version
cursor-agent --version
gh auth status             # already authenticated via the built-in token
git config user.email      # from dotfiles, or set it now
```

Confirm Cursor is authenticated:

```bash
cursor-agent status
```

If it reports "not authenticated", `CURSOR_API_KEY` did not arrive. Check:

```bash
echo "${CURSOR_API_KEY:0:5}"   # should print "crsr_", not empty
```

Empty means the secret is not scoped to this repo, or the Codespace was started before the secret was created. Fix the scope at github.com/settings/codespaces, then **stop and restart** the Codespace (a rebuild is not required; the secret is injected at start).

---

## Step 7 — First run with the Cursor agent

`cursor-agent` has three modes. Use them deliberately; the difference between them is the difference between a useful session and a mess.

```bash
cd /workspaces/cryptarium

# Interactive session — the default working mode
cursor-agent

# One-shot, non-interactive (scriptable, CI-friendly)
cursor-agent -p "Summarize the detector interface in internal/detector and list what is unimplemented"

# Resume the previous session
cursor-agent resume
```

**Ask → Plan → Agent.** For anything non-trivial in this codebase, ask for a plan before allowing edits. The architecture in `DESIGN.md` has narrow stage contracts, and an agent that jumps straight to code tends to blur them. A good opening prompt:

```
Read DESIGN.md and AGENT.md. We are on Phase 1 of the build plan:
the certificate/key detector and the dependency-manifest detector.
Propose a plan for internal/detector/certs that satisfies the Detector
interface in internal/detector/detector.go. Do not write code yet.
```

The agent reads `AGENTS.md` automatically from the repo root, so the project conventions, the "never invent a CVE or a NIST claim" rule, and the phase boundaries are in its context on every invocation. That file is the main lever you have over agent behavior; when the agent does something wrong twice, fix `AGENT.md` rather than re-explaining it in chat.

---

## Step 8 — Daily workflow

```bash
# Start where you left off
gh codespace list
gh codespace ssh            # or reopen the browser tab / VS Code

# Normal loop
make lint test              # or: golangci-lint run && gotestsum ./...
cursor-agent                # delegate the next unit of work
git add -p && git commit
git push
gh pr create --fill
```

**Lifecycle facts worth knowing:**

| Behavior | Detail |
|---|---|
| Idle timeout | Default 30 min, then the Codespace stops. Configurable in settings. |
| Stopped Codespace | Disk persists. `/workspaces` contents, uncommitted changes, and shell history survive. |
| Retention | Deleted automatically after 30 days of inactivity by default. **Push your work.** |
| Container rebuild | Command palette → *Codespaces: Rebuild Container*. Needed after editing `devcontainer.json`. Use *Full Rebuild* to bypass the image cache. |
| Home directory | `~` is **not** guaranteed to survive a rebuild. Anything you need permanently belongs in `post-create.sh` or dotfiles, never in a one-off `~` install. |
| Billing | Core-hours accrue while running; storage accrues while it exists. Stop it when you walk away: `gh codespace stop`. |

**Prebuilds.** Once the container config settles, turn on a prebuild (repo **Settings → Codespaces → Set up prebuild**, branch `main`). It bakes the built image on every push to `main`, dropping new-Codespace start time from minutes to seconds. Worth doing before you start scanning large fixture corpora, when you will be creating and discarding Codespaces frequently.

---

## Step 9 — Port forwarding (later phases)

Not needed for the CLI, but relevant once you build the HTML report viewer or a local dashboard:

```bash
go run ./cmd/cryptarium report --serve --port 8080
```

Codespaces auto-forwards the port and offers a URL in the **Ports** tab. Ports are private to you by default; use **Port Visibility → Public** only for a demo, and remember that a public forwarded port is a public URL.

---

## Step 10 — Troubleshooting

| Symptom | Cause | Fix |
|---|---|---|
| `cursor-agent: command not found` | Install ran before `~/.local/bin` was on `PATH`, or it failed | `bash .devcontainer/post-create.sh` to re-run; check the creation log |
| Cursor says "not authenticated" | Secret missing or out of scope | See Step 6; restart the Codespace after fixing |
| `devcontainer.json` edits have no effect | Config is read at container create/rebuild | Rebuild Container |
| CGO / tree-sitter build failure | Missing `build-essential` | It is installed by `post-create.sh`; confirm with `gcc --version` |
| Git push rejected | Default Codespaces token scope | `gh auth refresh -h github.com -s repo,workflow` |
| Everything is slow | 2-core machine under a concurrent scan | Change machine type to 4-core; it rebuilds onto a new VM |
| Container build fails on a feature | Upstream feature version pinned to a moving tag | Check the build log's failing step; pin the feature version explicitly |

---

## What you end up with

A repo whose environment is fully described in code: anyone (including future you, on a different laptop, in 90 seconds) gets an identical Go 1.26 toolchain, linters, `cursor-agent`, and project conventions by opening a Codespace. The laptop holds nothing but a browser session. That reproducibility is the same property the tool itself promises about scan output — worth getting right in your own development environment first.
