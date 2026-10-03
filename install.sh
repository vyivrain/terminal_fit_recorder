#!/usr/bin/env bash
#
# install.sh — build terminal_fit_recorder from source and install it into the
# current user's profile (no sudo), so it is available in every shell.
#
# Works on macOS and Linux. Builds natively for the current machine, so the
# CGO-based sqlite driver (github.com/mattn/go-sqlite3) compiles against the
# host C toolchain — no cross-compilation headaches.
#
# Usage:
#   ./install.sh                      # build + install to ~/.local/bin
#   INSTALL_DIR=~/bin ./install.sh    # install somewhere else on your PATH
#   BINARY_NAME=tfr ./install.sh      # install under a shorter command name
#
# Your workout data lives in ~/.terminal_fit_recorder/ and is never touched by
# this script; the freshly installed binary runs any pending DB migrations on
# first launch.

set -euo pipefail

# --- resolve repo root (so the script works from any directory) --------------
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# --- configuration -----------------------------------------------------------
BINARY_NAME="${BINARY_NAME:-terminal_fit_recorder}"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
SRC_PKG="./cmd"

info()  { printf '\033[0;34m==>\033[0m %s\n' "$*"; }
ok()    { printf '\033[0;32m✓\033[0m %s\n' "$*"; }
warn()  { printf '\033[0;33m!\033[0m %s\n' "$*"; }
die()   { printf '\033[0;31m✗ %s\033[0m\n' "$*" >&2; exit 1; }

# --- preflight ---------------------------------------------------------------
command -v go >/dev/null 2>&1 || die "Go is not installed or not on PATH. Install it from https://go.dev/dl/ and retry."

OS="$(uname -s)"
ARCH="$(uname -m)"
info "Building $BINARY_NAME natively for $OS/$ARCH ..."

# --- build (native, CGO on for sqlite) ---------------------------------------
TMP_BIN="$(mktemp -t "${BINARY_NAME}.XXXXXX")"
trap 'rm -f "$TMP_BIN"' EXIT

CGO_ENABLED=1 go build -o "$TMP_BIN" "$SRC_PKG" \
  || die "Build failed. Ensure a C compiler is available (Xcode CLT on macOS: 'xcode-select --install'; gcc on Linux)."
ok "Build succeeded"

# --- install (atomic move into place) ----------------------------------------
mkdir -p "$INSTALL_DIR"
DEST="$INSTALL_DIR/$BINARY_NAME"
mv -f "$TMP_BIN" "$DEST"
chmod +x "$DEST"
trap - EXIT
ok "Installed to $DEST"

# --- ensure INSTALL_DIR is on PATH -------------------------------------------
on_path=false
case ":$PATH:" in
  *":$INSTALL_DIR:"*) on_path=true ;;
esac

if $on_path; then
  ok "$INSTALL_DIR is already on your PATH"
else
  shell_name="$(basename "${SHELL:-}")"
  marker="# added by terminal_fit_recorder install.sh"
  added_to=""

  add_line() {  # $1 = rc file, $2 = line to append
    local rc="$1" line="$2"
    [ -f "$rc" ] || touch "$rc"
    if ! grep -qF "$marker" "$rc" 2>/dev/null; then
      printf '\n%s\n%s\n' "$marker" "$line" >> "$rc"
      added_to="$rc"
    else
      added_to="$rc"  # marker already present; treat as handled
    fi
  }

  case "$shell_name" in
    zsh)  add_line "$HOME/.zshrc"  "export PATH=\"$INSTALL_DIR:\$PATH\"" ;;
    bash)
      add_line "$HOME/.bashrc" "export PATH=\"$INSTALL_DIR:\$PATH\""
      # macOS login shells read .bash_profile, not .bashrc — cover both.
      if [ "$OS" = "Darwin" ]; then
        add_line "$HOME/.bash_profile" "export PATH=\"$INSTALL_DIR:\$PATH\""
      fi
      ;;
    fish) add_line "$HOME/.config/fish/config.fish" "fish_add_path \"$INSTALL_DIR\"" ;;
    *)    add_line "$HOME/.profile" "export PATH=\"$INSTALL_DIR:\$PATH\"" ;;
  esac

  warn "$INSTALL_DIR was not on your PATH — added it to $added_to"
  warn "Open a new terminal, or run:  source \"$added_to\""
fi

echo
ok "Done! Run:  $BINARY_NAME"
