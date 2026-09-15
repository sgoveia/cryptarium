#!/usr/bin/env bash
# Cryptarium dev container — per-start checks. Must stay fast.
set -uo pipefail

export PATH="$HOME/.local/bin:$HOME/go/bin:$PATH"

if [[ -z "${CURSOR_API_KEY:-}" ]]; then
  cat <<'MSG'

  !  CURSOR_API_KEY is not set in this Codespace.

     cursor-agent will not be able to authenticate. Fix:
       1. https://github.com/settings/codespaces -> Codespaces secrets
       2. New secret: CURSOR_API_KEY  (value starts with crsr_)
       3. Grant it access to this repository
       4. Stop and restart this Codespace (secrets are injected at start)

MSG
fi

# Warm the module cache in the background; never block the shell on it.
[[ -f go.mod ]] && (go mod download >/dev/null 2>&1 &)

echo "cryptarium dev container ready - 'make help' for targets, 'cursor-agent' to start an agent session."
