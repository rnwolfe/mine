#!/usr/bin/env bash
# cloud-setup.sh — Bootstrap Claude Code Cloud sessions with required tooling.
# Called via SessionStart hook in .claude/settings.json.
set -euo pipefail

# Only run in remote/cloud environments.
if [ "${CLAUDE_CODE_REMOTE:-}" != "true" ]; then
  exit 0
fi

GO_VERSION="1.25.5"

# Derive OS/arch for Go tarball (defaulting to linux/amd64 for unknown cases).
GO_OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
if command -v dpkg >/dev/null 2>&1; then
  _arch="$(dpkg --print-architecture)"
else
  _arch="$(uname -m)"
fi
case "${_arch}" in
  x86_64|amd64)
    GO_ARCH="amd64"
    ;;
  aarch64|arm64)
    GO_ARCH="arm64"
    ;;
  *)
    # Fallback to amd64 to preserve previous behavior on unexpected architectures.
    GO_ARCH="amd64"
    ;;
esac

GO_TARBALL="go${GO_VERSION}.${GO_OS}-${GO_ARCH}.tar.gz"
GO_URL="https://go.dev/dl/${GO_TARBALL}"

export PATH="/usr/local/go/bin:${HOME}/go/bin:${PATH}"

# --- Go ---
install_go() {
  if go version 2>/dev/null | grep -q "go${GO_VERSION}"; then
    return 0
  fi
  echo "Installing Go ${GO_VERSION}..."
  curl -sSL "${GO_URL}" -o "/tmp/${GO_TARBALL}"
  rm -rf /usr/local/go
  tar -C /usr/local -xzf "/tmp/${GO_TARBALL}"
  rm -f "/tmp/${GO_TARBALL}"
  echo "Go $(go version) installed."
}

# --- gh CLI ---
install_gh() {
  if command -v gh &>/dev/null; then
    return 0
  fi
  echo "Installing gh CLI..."
  curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg \
    | gpg --dearmor -o /usr/share/keyrings/githubcli-archive-keyring.gpg 2>/dev/null
  echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" \
    | tee /etc/apt/sources.list.d/github-cli.list >/dev/null
  apt-get update -qq && apt-get install -y -qq gh >/dev/null
  echo "gh $(gh --version | head -1) installed."
}

# --- Persist PATH for subsequent Bash tool calls ---
persist_env() {
  if [ -n "${CLAUDE_ENV_FILE:-}" ]; then
    {
      echo "PATH=/usr/local/go/bin:${HOME}/go/bin:${PATH}"
      echo "GOPATH=${HOME}/go"
    } >> "${CLAUDE_ENV_FILE}"
  fi
}

# --- Build mine ---
build_mine() {
  if [ -f "${CLAUDE_PROJECT_DIR:-}/Makefile" ]; then
    echo "Building mine..."
    make -C "${CLAUDE_PROJECT_DIR}" build
    echo "mine built."
  fi
}

install_go
install_gh
persist_env
build_mine

echo "Cloud environment ready."
