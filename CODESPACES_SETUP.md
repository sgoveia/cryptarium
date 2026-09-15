# Cryptarium — GitHub Codespaces Setup Guide

A step-by-step path from an empty GitHub account to a working `cryptarium` development environment in Codespaces, driven entirely from a laptop browser and a local terminal with standard git, with the Cursor CLI agent running inside the container.

**Assumptions**

- All code lives in one GitHub repo: `cryptarium`.
- The laptop is a thin client. No Go toolchain, no Cursor IDE, no build tooling installed locally. The only local requirements are a browser, `git`, and (optionally) VS Code Desktop.
- No `gh` CLI is installed locally — every operation that could use it is done either through the GitHub web UI or plain `git` commands.
- Development is done with the Cursor CLI (`cursor-agent`) running **inside** the Codespace, in the integrated terminal.

**Time to first green build:** about 20 minutes, most of it waiting on the first container build.

---

## Step 0 — Accounts and one-time prerequisites

| Item | Why | Where |
|---|---|---|
| GitHub account with Codespaces enabled | The dev environment | Free tier includes monthly core-hours; a paid plan is worth it for a 4-core machine |
| Cursor account with CLI access | `cursor-agent` requires a logged-in Cursor plan | https://cursor.com |
| Cursor API key | Headless auth inside the Codespace | Cursor dashboard → Settings / API keys → create key (starts `crsr_...`) |
| `git` on the laptop | Clone the repo and push files | Ships with macOS; `winget install Git.Git` on Windows; `apt install git` on Linux |
| (Optional) VS Code Desktop + "GitHub Codespaces" extension | Nicer editor than the browser tab; same remote container | https://code.visualstudio.com |

Generate the Cursor API key now and keep it on the clipboard. Step 3 consumes it.

---

## Step 1 — Create the `cryptarium` repository

Create the repo through the GitHub web UI:

1. Go to **https://github.com/new**.
2. Repository name: `cryptarium`.
3. Visibility: **Public**.
4. Check **Add a README file** (creates the initial commit so the repo is immediately cloneable).
5. Add `.gitignore` template: **Go**.
6. Choose a license: **Apache License 2.0** — the patent grant matters for cryptographic tooling, and it is easier to set this at creation than to add it later.
7. Click **Create repository**.

Then clone it locally:

```bash
git clone https://github.com/<your-username>/cryptarium.git
cd cryptarium
```

If your account uses SSH keys:

```bash
git clone git@github.com:<your-username>/cryptarium.git
cd cryptarium
```

---

## Step 2 — Seed the repo with the config and docs

Copy the following files into the local clone. Their contents come from the companion artifacts in this set (`DEVCONTAINER_CONFIG.md` for the `.devcontainer/` files; the `README.md`, `DESIGN.md`, and `AGENT.md` artifacts directly). The fastest path is to use `bootstrap-cryptarium.sh`, which creates every missing file in one pass — see the note at the end of this step.

```
cryptarium/
├── .devcontainer/
│   ├── devcontainer.json
│   ├── post-create.sh       ← must be executable
│   └── post-start.sh        ← must be executable
├── .github/
│   └── workflows/
│       └── ci.yml
├── .gitignore               ← already created by GitHub; replace or merge
├── .golangci.yml
├── AGENT.md
├── AGENTS.md                ← copy of AGENT.md (Cursor auto-discovers this name)
├── DESIGN.md
├── Makefile
└── README.md
```

Make the lifecycle scripts executable before committing — git tracks the executable bit and a non-executable `post-create.sh` is the most common silent devcontainer failure:

```bash
chmod +x .devcontainer/post-create.sh .devcontainer/post-start.sh
git update-index --chmod=+x .devcontainer/post-create.sh .devcontainer/post-start.sh
```

Commit and push everything:

```bash
git add .
git commit -m "chore: devcontainer, CI, agent instructions, design brief"
git push origin main
```

> **Why this order matters.** A Codespace is created *from a commit*. If `.devcontainer/` is not pushed before you launch, GitHub uses the default universal image and none of the toolchain is installed. Push first, launch second.

**Using `bootstrap-cryptarium.sh`.** The bootstrap script (also in this artifact set) audits the repo and creates every missing infrastructure file in a single run, sets executable bits, and copies `AGENT.md` to `AGENTS.md`. Run it from the repo root after copying in the three authored docs (`README.md`, `DESIGN.md`, `AGENT.md`):

```bash
# Copy bootstrap-cryptarium.sh into the repo root, then:
bash bootstrap-cryptarium.sh --check    # dry run — report only
bash bootstrap-cryptarium.sh            # create missing files
git add .
git commit -m "chore: devcontainer, CI, agent instructions, design brief"
git push origin main
```

---

## Step 3 — Configure Codespaces secrets

The Codespace is headless, so `cursor-agent` cannot open a browser to complete an interactive login. Supply the API key as an encrypted secret instead. It is injected as an environment variable into every Codespace for this repo on every start, with no key ever committed to git.

1. Go to **https://github.com/settings/codespaces**.
2. Under **Codespaces secrets**, click **New secret**.
3. Name: `CURSOR_API_KEY`. Value: the `crsr_...` key from Step 0.
4. Repository access: select `cryptarium` (or "All repositories" if you prefer).

Add these while you are there if you want them:

| Secret | Purpose |
|---|---|
| `CURSOR_API_KEY` | Cursor CLI headless auth — **required** |
| `ANTHROPIC_API_KEY` | For the optional AI triage layer in §8 of the design brief, once you build it |

> A secret change takes effect on the **next** Codespace start. If a Codespace is already running, stop it and start it again — a full rebuild is not required.

---

## Step 4 — (Optional but recommended) Set up a personal dotfiles repo

`.devcontainer/` is project configuration: everyone who opens `cryptarium` gets it. Dotfiles are *personal* configuration — your shell, aliases, git identity, and editor preferences — applied automatically to **every** Codespace you create, in any repo.

1. Create a public repo named `dotfiles` on your account (github.com/new, name `dotfiles`, Public, with a README).
2. Clone it locally, add the files from `DEVCONTAINER_CONFIG.md` (section "Dotfiles repository"), commit, and push.
3. Go to **https://github.com/settings/codespaces** → **Dotfiles** → check **Automatically install dotfiles**.

```bash
git clone https://github.com/<your-username>/dotfiles.git
cd dotfiles
# add install.sh, .zshrc, .gitconfig, .config/cursor-agent/cli-config.json
git add .
git commit -m "chore: initial dotfiles"
git push origin main
```

Codespaces clones the repo into `~/dotfiles` and runs `install.sh` after the container is created. Keep it fast and idempotent — a slow or failing dotfiles script silently degrades every Codespace you launch.

**Rule of thumb for what goes where:**

| Goes in `.devcontainer/` | Goes in `dotfiles` |
|---|---|
| Go toolchain, linters, `cursor-agent` | Shell prompt, aliases, `.zshrc` |
| Anything a contributor needs to build the project | `git` name/email, `git` aliases |
| Anything CI should also have | Personal `cursor-agent` defaults |

---

## Step 5 — Launch the Codespace

Open the repo on GitHub, click **Code ▾ → Codespaces → Create codespace on main**.

On the creation dialog, click **Configure and create codespace** if you want to explicitly select the machine type. Choose **4-core** (`standardLinux32gb`): tree-sitter builds with CGO, and the fleet-scale scanning work in this project is concurrency-heavy, so you want cores to test against. 2-core works, but single-core timing is misleading. You can change the machine type later from the Codespaces page, which moves the environment onto a new VM without losing disk state.

The first build takes 3–6 minutes: GitHub pulls the base image, installs features, then runs `post-create.sh`. Watch the progress via **"Building codespace"** → **View log**. Read that log the first time — almost every setup problem is visible there in plain text.

To open the Codespace in VS Code Desktop instead of the browser tab, click the **"..."** menu on the Codespace card and choose **Open in Visual Studio Code**. The "GitHub Codespaces" extension handles the SSH tunnel; no configuration needed.

---

## Step 6 — Verify the environment

In the Codespace terminal (browser tab or VS Code):

```bash
go version                 # go1.26.x
golangci-lint --version
gotestsum --version
cursor-agent --version
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

Empty means the secret is not scoped to this repo, or the Codespace was started before the secret was created. Fix the scope at github.com/settings/codespaces, then **stop and restart** the Codespace from the browser (the **"..."** menu on the Codespace card → **Stop codespace**, then reopen it). A rebuild is not required; secrets are injected at start, not at build.

---

## Step 7 — First run with the Cursor agent

`cursor-agent` has three modes. Use them deliberately — the difference between them is the difference between a useful session and a mess.

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

The agent reads `AGENTS.md` automatically from the repo root, so the project conventions, the "never invent a NIST claim" rule, and the phase boundaries are in its context on every invocation. That file is the main lever you have over agent behavior — when the agent does something wrong twice, fix `AGENT.md` rather than re-explaining it in chat.

---

## Step 8 — Daily workflow

**Starting a session** — from the browser, go to github.com/codespaces and click the Codespace to resume it. In VS Code Desktop, the "GitHub Codespaces" extension lists your active Codespaces under the Remote Explorer sidebar — click to connect.

**The normal loop, all inside the Codespace terminal:**

```bash
cd /workspaces/cryptarium

make check                  # fmt + lint + test — same as CI
cursor-agent                # delegate the next unit of work
git add -p && git commit -m "feat(detector/certs): parse PKCS#12 bundles"
git push origin main
```

To open a pull request, push a branch and use the GitHub web UI:

```bash
git checkout -b feat/certs-detector
# ... work ...
git push origin feat/certs-detector
# Then: github.com/cryptarium → Pull requests → New pull request
```

**Lifecycle facts worth knowing:**

| Behavior | Detail |
|---|---|
| Idle timeout | Default 30 min, then the Codespace stops. Configurable at github.com/settings/codespaces → Default idle timeout. |
| Stopped Codespace | Disk persists. `/workspaces` contents, uncommitted changes, and shell history survive a stop/start. |
| Retention | Deleted automatically after 30 days of inactivity by default. **Push your work.** |
| Container rebuild | Command palette (`F1`) → *Codespaces: Rebuild Container*. Required after editing `devcontainer.json`. Use *Full Rebuild* to bypass the image cache if a layer is stale. |
| Home directory | `~` is **not** guaranteed to survive a rebuild. Anything you need permanently belongs in `post-create.sh` or dotfiles, never in a one-off `~` install. |
| Billing | Core-hours accrue while running; storage accrues while the Codespace exists. Stop it when you walk away — **Code ▾ → Manage Codespaces → Stop** on the repo page. |

**Prebuilds.** Once the container config settles, turn on a prebuild: repo **Settings → Codespaces → Set up prebuild**, branch `main`. It bakes the built image on every push to `main`, dropping new-Codespace start time from minutes to seconds. Worth doing before you start scanning large fixture corpora, when you will be creating and discarding Codespaces frequently.

---

## Step 9 — Port forwarding (later phases)

Not needed for the CLI, but relevant once you build the HTML report viewer or a local dashboard:

```bash
go run ./cmd/cryptarium report --serve --port 8080
```

Codespaces auto-forwards the port and makes a URL available under the **Ports** tab in the editor. Ports are private to you by default. Use **Port Visibility → Public** only for a demo, and remember that a public forwarded port is a publicly accessible URL.

---

## Step 10 — Troubleshooting

| Symptom | Cause | Fix |
|---|---|---|
| `cursor-agent: command not found` | Install ran before `~/.local/bin` was on `PATH`, or the install failed | `bash .devcontainer/post-create.sh` to re-run; check the creation log in the browser |
| Cursor says "not authenticated" | `CURSOR_API_KEY` secret missing or not scoped to this repo | See Step 6; stop and restart the Codespace after fixing |
| `devcontainer.json` edits have no effect | Config is only read at container create/rebuild | Command palette → *Codespaces: Rebuild Container* |
| CGO / tree-sitter build failure | Missing `build-essential` | Installed by `post-create.sh`; confirm with `gcc --version` |
| `git push` asks for credentials | HTTPS remote without a credential helper | Set up a [Personal Access Token](https://github.com/settings/tokens) with `repo` scope and use it as the password, or switch the remote to SSH: `git remote set-url origin git@github.com:<user>/cryptarium.git` |
| `git push` rejected after adding a workflow file | Codespaces built-in token lacks `workflow` scope | Add a PAT with `repo` + `workflow` scopes and use it for this push, or push the workflow file from your local machine |
| Post-create script changes have no effect | `post-create.sh` only runs on create/rebuild, not on start | Run it manually: `bash .devcontainer/post-create.sh` |
| Everything is slow | 2-core machine under a concurrent scan | Stop the Codespace, change machine type to 4-core from github.com/codespaces, restart |
| Container build fails on a feature | Upstream feature version pinned to a moving tag | Check the creation log's failing step; pin the feature version explicitly in `devcontainer.json` |
| Codespace disappeared | Exceeded the 30-day inactivity retention limit | Re-create from the latest commit; this is why pushing regularly matters |

---

## What you end up with

A repo whose environment is fully described in code: anyone (including future you, on a different laptop, in under 90 seconds) gets an identical Go 1.26 toolchain, linters, `cursor-agent`, and project conventions by opening a Codespace. The laptop holds nothing but a browser and a standard `git` install. That reproducibility is the same property the tool itself promises about scan output — worth getting right in your own development environment first.
