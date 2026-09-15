#!/usr/bin/env bash
# Cryptarium dev container — one-time setup.
# Idempotent: safe to re-run with `bash .devcontainer/post-create.sh`.
set -euo pipefail

log()  { printf '\033[1;36m[setup]\033[0m %s\n' "$*"; }
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

mkdir -p "$HOME/.local/bin"
# Debian ships fd as fdfind; alias the conventional name.
if have fdfind && ! have fd; then
  ln -sf "$(command -v fdfind)" "$HOME/.local/bin/fd"
fi

export PATH="$HOME/.local/bin:$HOME/go/bin:$PATH"

# ---------------------------------------------------------------------------
# 2. Cursor CLI
#    Installs `cursor-agent` into ~/.local/bin. Auth comes from the
#    CURSOR_API_KEY Codespaces secret; no interactive browser login needed.
# ---------------------------------------------------------------------------
if have cursor-agent; then
  log "Cursor CLI already present: $(cursor-agent --version 2>/dev/null || echo unknown)"
else
  log "Installing Cursor CLI"
  curl https://cursor.com/install -fsS | bash || \
    echo "[setup] WARNING: Cursor CLI install failed; re-run this script later." >&2
fi

# ---------------------------------------------------------------------------
# 3. Go tooling
# ---------------------------------------------------------------------------
log "Installing Go tooling"
GO_TOOLS=(
  "github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest"  # lint
  "gotest.tools/gotestsum@latest"                                  # test output
  "mvdan.cc/gofumpt@latest"                                        # strict gofmt
  "golang.org/x/tools/cmd/goimports@latest"                        # imports
  "golang.org/x/vuln/cmd/govulncheck@latest"                       # vuln scan
  "github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@latest" # SBOM of ourselves
  "github.com/goreleaser/goreleaser/v2@latest"                     # release binaries
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

# Certificate fixtures: a deliberately quantum-vulnerable RSA-2048 cert and an
# ECDSA P-256 cert. Used only by tests under testdata/.
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
