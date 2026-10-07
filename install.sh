#!/bin/sh
# termidi installer: curl -fsSL https://raw.githubusercontent.com/EmreErdogan/termidi/main/install.sh | sh
set -eu

APP=termidi
REPO=EmreErdogan/termidi
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"

os=$(uname -s | tr '[:upper:]' '[:lower:]')
arch=$(uname -m)
case "$arch" in
  x86_64|amd64) arch=amd64 ;;
  arm64|aarch64) arch=arm64 ;;
  *) echo "unsupported architecture: $arch" >&2; exit 1 ;;
esac
case "$os" in
  darwin|linux) ;;
  *) echo "unsupported OS: $os" >&2; exit 1 ;;
esac

url="https://github.com/$REPO/releases/latest/download/$APP-$os-$arch"
mkdir -p "$INSTALL_DIR"
echo "downloading $url"
curl -fsSL "$url" -o "$INSTALL_DIR/$APP.tmp"
chmod 755 "$INSTALL_DIR/$APP.tmp"
mv "$INSTALL_DIR/$APP.tmp" "$INSTALL_DIR/$APP"

# Make sure INSTALL_DIR is on PATH for future shells.
case ":$PATH:" in
  *":$INSTALL_DIR:"*) ;;
  *)
    line="export PATH=\"$INSTALL_DIR:\$PATH\""
    case "${SHELL:-}" in
      */zsh) rc="$HOME/.zshrc" ;;
      */bash) rc="$HOME/.bashrc"; [ "$os" = darwin ] && rc="$HOME/.bash_profile" ;;
      */fish) rc="$HOME/.config/fish/config.fish"; line="fish_add_path $INSTALL_DIR" ;;
      *) rc="$HOME/.profile" ;;
    esac
    mkdir -p "$(dirname "$rc")"
    if ! grep -qsF "$INSTALL_DIR" "$rc"; then
      printf '\n# %s\n%s\n' "$APP" "$line" >> "$rc"
      echo "added $INSTALL_DIR to PATH in $rc (open a new terminal or run: source $rc)"
    fi
    ;;
esac

echo "installed $INSTALL_DIR/$APP ($("$INSTALL_DIR/$APP" version))"
