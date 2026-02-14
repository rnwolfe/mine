#!/usr/bin/env bash
# cloud-setup.sh — Bootstrap Claude Code Cloud sessions with required tooling.
# Called via SessionStart hook in .claude/settings.json.
set -euo pipefail

# Only run in remote/cloud environments.
if [ "${CLAUDE_CODE_REMOTE:-}" != "true" ]; then
  exit 0
fi

GO_VERSION="1.25.5"

# Derive OS/arch for Go tarball.
GO_OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
if command -v dpkg >/dev/null 2>&1; then
  _arch="$(dpkg --print-architecture)"
else
  _arch="$(uname -m)"
fi
case "${_arch}" in
  x86_64|amd64)  GO_ARCH="amd64" ;;
  aarch64|arm64)  GO_ARCH="arm64" ;;
  *)
    echo "cloud-setup: unsupported architecture '${_arch}' for Go installation" >&2
    exit 1
    ;;
esac

GO_TARBALL="go${GO_VERSION}.${GO_OS}-${GO_ARCH}.tar.gz"
GO_URL="https://go.dev/dl/${GO_TARBALL}"

export PATH="/usr/local/go/bin:${HOME}/go/bin:${PATH}"

# Use sudo if available and not already root.
_sudo() {
  if [ "$(id -u)" -eq 0 ]; then
    "$@"
  elif command -v sudo &>/dev/null; then
    sudo "$@"
  else
    echo "cloud-setup: need root for: $*" >&2
    return 1
  fi
}

# --- Go ---
install_go() {
  if go version 2>/dev/null | grep -q "go${GO_VERSION}"; then
    return 0
  fi
  echo "Installing Go ${GO_VERSION} (${GO_OS}/${GO_ARCH})..."
  curl -sSL "${GO_URL}" -o "/tmp/${GO_TARBALL}"
  _sudo rm -rf /usr/local/go
  _sudo tar -C /usr/local -xzf "/tmp/${GO_TARBALL}"
  rm -f "/tmp/${GO_TARBALL}"
  echo "Go $(go version) installed."
}

# --- gh CLI ---
install_gh() {
  if command -v gh &>/dev/null; then
    return 0
  fi
  if ! command -v apt-get &>/dev/null; then
    echo "cloud-setup: apt-get not found — gh CLI install requires a Debian/Ubuntu base" >&2
    return 1
  fi
  echo "Installing gh CLI..."
  local gpg_err="/tmp/cloud-setup-gpg-$$.log"
  if ! curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg \
    | _sudo gpg --dearmor -o /usr/share/keyrings/githubcli-archive-keyring.gpg 2>"${gpg_err}"; then
    echo "cloud-setup: failed to import GitHub CLI archive key" >&2
    [ -s "${gpg_err}" ] && cat "${gpg_err}" >&2
    rm -f "${gpg_err}"
    return 1
  fi
  rm -f "${gpg_err}"
  echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" \
    | _sudo tee /etc/apt/sources.list.d/github-cli.list >/dev/null
  _sudo apt-get update -qq && _sudo apt-get install -y -qq gh >/dev/null
  echo "gh $(gh --version | head -1) installed."
}

# --- Persist PATH for subsequent Bash tool calls (idempotent) ---
persist_env() {
  if [ -n "${CLAUDE_ENV_FILE:-}" ]; then
    # Remove any existing PATH/GOPATH lines to avoid unbounded growth.
    if [ -f "${CLAUDE_ENV_FILE}" ]; then
      grep -vE '^(PATH=|GOPATH=)' "${CLAUDE_ENV_FILE}" > "${CLAUDE_ENV_FILE}.tmp" || true
      mv "${CLAUDE_ENV_FILE}.tmp" "${CLAUDE_ENV_FILE}"
    fi
    {
      # Use single quotes so $PATH expands at source-time, not write-time.
      echo 'PATH=/usr/local/go/bin:${HOME}/go/bin:$PATH'
      echo 'GOPATH=${HOME}/go'
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
