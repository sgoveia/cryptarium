# Cryptarium — Dev Container and Dotfiles Configuration

Everything needed to make a Codespace for `cryptarium` reproducible: Go toolchain, Cursor CLI, linters, crypto/CBOM tooling, and personal shell setup that survives every rebuild.

**File map**

| Path | Lives in | Purpose |
|---|---|---|
| `.devcontainer/devcontainer.json` | `cryptarium` repo | Container image, features, extensions, lifecycle hooks |
| `.devcontainer/post-create.sh` | `cryptarium` repo | One-time setup: Cursor CLI, Go tooling, build deps |
| `.devcontainer/post-start.sh` | `cryptarium` repo | Every-start: auth check, module download |
| `.devcontainer/README.md` | `cryptarium` repo | Short note for contributors |
| `dotfiles/install.sh` + rc files | personal `dotfiles` repo | Shell, aliases, git identity — applies to all your Codespaces |

**The rule that governs the split:** if a contributor needs it to build the project, it goes in `.devcontainer/`. If only you want it, it goes in `dotfiles`. Nothing important should live in a one-off `~` install, because `~` does not reliably survive a container rebuild.

---

## 1. `.devcontainer/devcontainer.json`

```jsonc
{
  "name": "cryptarium",

  // Go 1.x on Debian bookworm. The image ships git, curl, build-essential,
  // and a non-root "vscode" user. Pin the minor line when you want
  // byte-identical builds: mcr.microsoft.com/devcontainers/go:1.26-bookworm
  "image": "mcr.microsoft.com/devcontainers/go:1-bookworm",

  "features": {
    // Zsh + common shell utilities, configured for the non-root user.
    "ghcr.io/devcontainers/features/common-utils:2": {
      "installZsh": true,
      "configureZshAsDefaultShell": true,
      "installOhMyZsh": true,
      "upgradePackages": true
    },
    // gh is used for PRs, SARIF upload checks, and repo enumeration work later.
    "ghcr.io/devcontainers/features/github-cli:1": {},
    // Node is not needed by cryptarium itself, but is needed to generate
    // realistic package-lock.json fixtures for the dependency detector.
    "ghcr.io/devcontainers/features/node:1": {
      "version": "lts"
    }
    // Uncomment when you reach the container-image scanning roadmap item.
    // "ghcr.io/devcontainers/features/docker-in-docker:2": {}
  },

  // 4 cores is the practical floor: tree-sitter builds with CGO and the
  // collector is concurrency-heavy, so single-core timing tells you nothing.
  "hostRequirements": {
    "cpus": 4,
    "memory": "8gb",
    "storage": "32gb"
  },

  "customizations": {
    "vscode": {
      "extensions": [
        "golang.go",
        "ms-vscode.makefile-tools",
        "redhat.vscode-yaml",
        "tamasfe.even-better-toml",
        "eamodio.gitlens",
        "github.vscode-github-actions",
        "streetsidesoftware.code-spell-checker",
        "bierner.markdown-mermaid"
      ],
      "settings": {
        "go.toolsManagement.autoUpdate": false,
        "go.lintTool": "golangci-lint",
        "go.lintOnSave": "package",
        "go.useLanguageServer": true,
        "gopls": {
          "ui.semanticTokens": true,
          "formatting.gofumpt": true
        },
        "editor.formatOnSave": true,
        "[go]": {
          "editor.defaultFormatter": "golang.go",
          "editor.codeActionsOnSave": { "source.organizeImports": "explicit" }
        },
        "files.insertFinalNewline": true,
        "files.trimTrailingWhitespace": true,
        "terminal.integrated.defaultProfile.linux": "zsh"
      }
    }
  },

  // Secrets configured at github.com/settings/codespaces are injected as
  // environment variables at container START. Listing them here documents
  // the contract and makes a missing one obvious.
  "remoteEnv": {
    "CURSOR_API_KEY": "${localEnv:CURSOR_API_KEY}",
    "ANTHROPIC_API_KEY": "${localEnv:ANTHROPIC_API_KEY}",
    "PATH": "${containerEnv:PATH}:/home/vscode/.local/bin:/home/vscode/go/bin",
    "GOFLAGS": "-buildvcs=false",
    "CRYPTARIUM_DEV": "1"
  },

  // Persist the Go module and build caches across rebuilds. Without this,
  // every rebuild re-downloads the dependency graph.
  "mounts": [
    "source=cryptarium-gomod,target=/home/vscode/go/pkg/mod,type=volume",
    "source=cryptarium-gocache,target=/home/vscode/.cache/go-build,type=volume"
  ],

  "onCreateCommand": "sudo chown -R vscode:vscode /home/vscode/go /home/vscode/.cache || true",
  "postCreateCommand": "bash .devcontainer/post-create.sh",
  "postStartCommand": "bash .devcontainer/post-start.sh",

  "remoteUser": "vscode",

  "portsAttributes": {
    "8080": { "label": "cryptarium report server", "onAutoForward": "notify" }
  }
}
```

**Notes on the choices**

- **`image` over `build`/Dockerfile.** A prebuilt devcontainer image plus features is cached by GitHub and starts faster than a custom Dockerfile. Move to a Dockerfile only when `post-create.sh` grows past ~30 seconds and you want it baked into a prebuild layer.
- **`remoteEnv` PATH.** `cursor-agent` installs to `~/.local/bin`; `go install` puts binaries in `~/go/bin`. Both need to be on `PATH` for non-interactive shells, which is where agent tool calls run. Setting it here rather than in `.zshrc` is what makes `cursor-agent` work when invoked from a script.
- **Named volume mounts.** These outlive container rebuilds within the same Codespace. They do not travel to a *new* Codespace, which is what prebuilds are for.
- **`hostRequirements`.** Advisory: it sets the default machine offered at creation. It does not stop you choosing a smaller one.

---

## 2. `.devcontainer/post-create.sh`

Runs once, after the container is created (and after every rebuild). Idempotent by design, so re-running it by hand is always safe.

```bash
#!/usr/bin/env bash
# Cryptarium dev container — one-time setup.
# Idempotent: safe to re-run at any time with `bash .devcontainer/post-create.sh`.
set -euo pipefail

log() { printf '\033[1;36m[setup]\033[0m %s\n' "$*"; }
have() { command -v "$1" >/dev/null 2>&1; }

export DEBIAN_FRONTEND=noninteractive

# ---------------------------------------------------------------------------
# 1. System packages
#    build-essential + pkg-config: tree-sitter Go bindings require CGO.
#    openssl + gnutls-bin: generate certificate fixtures for the certs detector.
# ---------------------------------------------------------------------------
log "Installing system packages"
sudo apt-get update -qq
sudo apt-get install -y -qq --no-install-recommends \
  build-essential pkg-config ca-certificates curl git jq unzip \
  openssl gnutls-bin ripgrep fd-find shellcheck
sudo rm -rf /var/lib/apt/lists/*

# Debian ships fd as fdfind; alias the conventional name.
if have fdfind && ! have fd; then
  mkdir -p "$HOME/.local/bin"
  ln -sf "$(command -v fdfind)" "$HOME/.local/bin/fd"
fi

mkdir -p "$HOME/.local/bin"
export PATH="$HOME/.local/bin:$HOME/go/bin:$PATH"

# ---------------------------------------------------------------------------
# 2. Cursor CLI
#    Installs `cursor-agent` into ~/.local/bin. Authentication is via the
#    CURSOR_API_KEY Codespaces secret; no interactive browser login needed.
# ---------------------------------------------------------------------------
if have cursor-agent; then
  log "Cursor CLI already present: $(cursor-agent --version 2>/dev/null || echo unknown)"
else
  log "Installing Cursor CLI"
  curl https://cursor.com/install -fsS | bash || {
    echo "[setup] WARNING: Cursor CLI install failed; re-run this script later." >&2
  }
fi

# ---------------------------------------------------------------------------
# 3. Go tooling
#    Pinned where the project depends on behavior; @latest where it does not.
# ---------------------------------------------------------------------------
log "Installing Go tooling"
GO_TOOLS=(
  "github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest"  # lint
  "gotest.tools/gotestsum@latest"                                   # test output
  "mvdan.cc/gofumpt@latest"                                         # strict gofmt
  "golang.org/x/tools/cmd/goimports@latest"                         # imports
  "golang.org/x/vuln/cmd/govulncheck@latest"                        # vuln scan
  "github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@latest" # SBOM of ourselves
  "github.com/goreleaser/goreleaser/v2@latest"                      # release binaries
)
for tool in "${GO_TOOLS[@]}"; do
  name="${tool%@*}"; name="${name##*/}"
  if have "$name"; then
    log "  ${name} already installed"
  else
    log "  installing ${name}"
    go install "$tool" || echo "[setup] WARNING: failed to install ${tool}" >&2
  fi
done

# ---------------------------------------------------------------------------
# 4. Project bootstrap
# ---------------------------------------------------------------------------
if [[ -f go.mod ]]; then
  log "Downloading Go modules"
  go mod download || true
fi

# Cursor discovers repo instructions as AGENTS.md; AGENT.md is the human name.
if [[ -f AGENT.md && ! -e AGENTS.md ]]; then
  log "Linking AGENT.md -> AGENTS.md for Cursor discovery"
  ln -sf AGENT.md AGENTS.md
fi

# Test fixtures for the certificate detector: a deliberately quantum-vulnerable
# RSA-2048 cert and an ECDSA P-256 cert. Never used outside testdata/.
if [[ -d testdata && ! -f testdata/certs/rsa2048.pem ]]; then
  log "Generating certificate fixtures"
  mkdir -p testdata/certs
  openssl req -x509 -newkey rsa:2048 -keyout testdata/certs/rsa2048.key \
    -out testdata/certs/rsa2048.pem -days 3650 -nodes \
    -subj "/CN=cryptarium-fixture-rsa2048" 2>/dev/null || true
  openssl req -x509 -newkey ec -pkeyopt ec_paramgen_curve:prime256v1 \
    -keyout testdata/certs/ecdsa-p256.key -out testdata/certs/ecdsa-p256.pem \
    -days 3650 -nodes -subj "/CN=cryptarium-fixture-p256" 2>/dev/null || true
fi

log "Setup complete."
log "  go:           $(go version 2>/dev/null || echo 'MISSING')"
log "  cursor-agent: $(cursor-agent --version 2>/dev/null || echo 'MISSING')"
log "  golangci:     $(golangci-lint --version 2>/dev/null | head -1 || echo 'MISSING')"
```

---

## 3. `.devcontainer/post-start.sh`

Runs on every container start, including resuming a stopped Codespace. Keep it under a second or two.

```bash
#!/usr/bin/env bash
# Cryptarium dev container — per-start checks. Must stay fast.
set -uo pipefail

export PATH="$HOME/.local/bin:$HOME/go/bin:$PATH"

if [[ -z "${CURSOR_API_KEY:-}" ]]; then
  cat <<'EOF'

  ⚠  CURSOR_API_KEY is not set in this Codespace.

     cursor-agent will not be able to authenticate. Fix:
       1. https://github.com/settings/codespaces → Codespaces secrets
       2. New secret: CURSOR_API_KEY  (value starts with crsr_)
       3. Grant it access to this repository
       4. Stop and restart this Codespace (secrets are injected at start)

EOF
fi

# Warm the module cache in the background; never block the shell on it.
[[ -f go.mod ]] && (go mod download >/dev/null 2>&1 &)

echo "cryptarium dev container ready — 'make help' for targets, 'cursor-agent' to start an agent session."
```

Make both executable before committing (a non-executable hook script is the most common silent devcontainer failure):

```bash
chmod +x .devcontainer/post-create.sh .devcontainer/post-start.sh
git update-index --chmod=+x .devcontainer/post-create.sh .devcontainer/post-start.sh
```

---

## 4. `.devcontainer/README.md`

```markdown
# Dev container

Opening this repo in a Codespace or in VS Code with the Dev Containers
extension gives you: Go 1.26, golangci-lint, gotestsum, gofumpt,
govulncheck, cyclonedx-gomod, goreleaser, the GitHub CLI, and the
Cursor CLI (`cursor-agent`).

- `devcontainer.json` — image, features, editor settings, lifecycle hooks
- `post-create.sh` — one-time setup; idempotent, safe to re-run
- `post-start.sh` — per-start checks

`cursor-agent` authenticates from the `CURSOR_API_KEY` Codespaces secret.
Contributors who do not use Cursor can ignore it; nothing in the build
depends on it.

After editing `devcontainer.json`: Command Palette →
*Codespaces: Rebuild Container*.
```

---

## 5. Dotfiles repository

Public repo named `dotfiles` on your account. Enable at **github.com/settings/codespaces → Dotfiles → Automatically install dotfiles**. Codespaces clones it to `~/dotfiles` and runs `install.sh`.

```
dotfiles/
├── install.sh
├── .zshrc
├── .gitconfig
└── .config/
    └── cursor-agent/
        └── cli-config.json
```

### `dotfiles/install.sh`

```bash
#!/usr/bin/env bash
# Personal dotfiles — runs in every Codespace after container creation.
# Must be fast and idempotent. A slow install.sh delays every environment
# you ever open; a failing one degrades them silently.
set -euo pipefail

DOTFILES="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

link() {
  local src="$DOTFILES/$1" dest="$HOME/$1"
  [[ -e "$src" ]] || return 0
  mkdir -p "$(dirname "$dest")"
  ln -sfn "$src" "$dest"
  echo "  linked $1"
}

echo "[dotfiles] linking"
link .zshrc
link .gitconfig
link .config/cursor-agent/cli-config.json

# Git identity — only if the environment has not already set it.
git config --global --get user.name  >/dev/null || git config --global user.name  "Stephen"
git config --global --get user.email >/dev/null || git config --global user.email "you@example.com"

git config --global init.defaultBranch main
git config --global pull.rebase true
git config --global rebase.autoStash true
git config --global push.autoSetupRemote true
git config --global diff.algorithm histogram
git config --global core.editor "code --wait"

echo "[dotfiles] done"
```

> Replace the email with the address on your GitHub account (or the `users.noreply.github.com` one) so commits attribute correctly.

### `dotfiles/.zshrc`

```bash
# ---- PATH ------------------------------------------------------------------
export PATH="$HOME/.local/bin:$HOME/go/bin:/usr/local/go/bin:$PATH"

# ---- Go --------------------------------------------------------------------
export GOPATH="$HOME/go"
export GOFLAGS="-buildvcs=false"
export CGO_ENABLED=1          # tree-sitter bindings need cgo

# ---- History ---------------------------------------------------------------
setopt HIST_IGNORE_ALL_DUPS INC_APPEND_HISTORY SHARE_HISTORY
HISTSIZE=50000; SAVEHIST=50000; HISTFILE="$HOME/.zsh_history"

# ---- General aliases -------------------------------------------------------
alias ll='ls -alh'
alias gs='git status -sb'
alias gd='git diff'
alias gco='git checkout'
alias gl='git log --oneline --graph --decorate -20'

# ---- Go aliases ------------------------------------------------------------
alias gt='gotestsum --format testname ./...'
alias gtr='gotestsum --format testname -- -race ./...'
alias gl8='golangci-lint run'
alias gfmt='gofumpt -l -w . && goimports -w .'
alias gvuln='govulncheck ./...'

# ---- Cursor agent ----------------------------------------------------------
alias ca='cursor-agent'
alias cap='cursor-agent -p'          # one-shot print mode
alias car='cursor-agent resume'      # resume last session

# ---- cryptarium ------------------------------------------------------------
alias cdc='cd /workspaces/cryptarium'
alias cr='go run ./cmd/cryptarium'
alias crs='go run ./cmd/cryptarium scan .'
# Self-scan: the tool must always be able to scan its own repo cleanly.
alias selfscan='go run ./cmd/cryptarium scan . --format markdown'

# Codespaces niceties
[[ -n "${CODESPACES:-}" ]] && export EDITOR="code --wait"
```

### `dotfiles/.gitconfig`

```ini
[alias]
    st   = status -sb
    co   = checkout
    br   = branch
    ci   = commit
    amend = commit --amend --no-edit
    lg   = log --graph --abbrev-commit --decorate --format=format:'%C(bold blue)%h%C(reset) %C(dim white)%an%C(reset) %C(bold green)(%ar)%C(reset)%C(auto)%d%C(reset) %s'
    undo = reset --soft HEAD~1
    wip  = "!git add -A && git commit -m 'wip: checkpoint'"

[core]
    excludesfile = ~/.gitignore_global

[fetch]
    prune = true
```

### `dotfiles/.config/cursor-agent/cli-config.json`

Personal agent defaults, applied in every Codespace. Keep permissions conservative; the repo-level `AGENT.md` is where behavioral rules belong.

```json
{
  "permissions": {
    "allow": [
      "Shell(go build*)",
      "Shell(go test*)",
      "Shell(go vet*)",
      "Shell(gotestsum*)",
      "Shell(golangci-lint*)",
      "Shell(gofumpt*)",
      "Shell(goimports*)",
      "Shell(git status*)",
      "Shell(git diff*)",
      "Shell(git log*)"
    ],
    "deny": [
      "Shell(git push*)",
      "Shell(git reset --hard*)",
      "Shell(gh pr merge*)",
      "Shell(rm -rf*)",
      "Shell(curl*)",
      "Read(.env)",
      "Read(**/*.key)",
      "Read(**/id_rsa*)"
    ]
  }
}
```

> The deny list is not paranoia for its own sake. This project's `testdata/` deliberately contains private keys and weak certificates, and an agent that reads or transmits them is doing exactly the wrong thing for a cryptographic-inventory tool. Fixture keys are generated locally, are never secret, and still should not leave the container.

---

## 6. `Makefile` (recommended companion)

The devcontainer's value doubles when both you and the agent have one command surface.

```makefile
.DEFAULT_GOAL := help
BIN := cryptarium
PKG := ./...

help: ## Show available targets
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?## "};{printf "  \033[36m%-14s\033[0m %s\n",$$1,$$2}'

build: ## Build the CLI
	go build -o bin/$(BIN) ./cmd/$(BIN)

test: ## Run tests with race detector
	gotestsum --format testname -- -race -coverprofile=coverage.out $(PKG)

lint: ## Lint
	golangci-lint run

fmt: ## Format
	gofumpt -l -w . && goimports -w .

vuln: ## Vulnerability scan
	govulncheck $(PKG)

selfscan: build ## Scan this repo with the tool itself
	./bin/$(BIN) scan . --format markdown

check: fmt lint test ## Everything CI runs

clean:
	rm -rf bin coverage.out

.PHONY: help build test lint fmt vuln selfscan check clean
```

---

## 7. Verification checklist

After a rebuild, all of these should pass:

```bash
go version                      # go1.26.x
cursor-agent --version          # installed
cursor-agent status             # authenticated
golangci-lint --version
gotestsum --version
govulncheck -version
cyclonedx-gomod version
gcc --version                   # CGO available for tree-sitter
openssl version                 # cert fixture generation
echo "${CURSOR_API_KEY:0:5}"    # crsr_
make help                       # targets listed
```

If any line fails, re-run `bash .devcontainer/post-create.sh` and read its output. If it still fails, the creation log (Codespaces page → **View creation log**) names the failing step directly.
