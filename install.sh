#!/bin/sh
# saka installer — curl -sL https://getsaka.dev/install.sh | sh
#
# NOTE: the source chat's own rendering of this script was corrupted —
# it displayed nested, duplicated `$(...)`/`${...}` substitutions (e.g.
# the download URL and the version lookup each appeared mangled and
# repeated several times over). That looks like a rendering bug in the
# chat page itself, not something to copy literally. This is a
# best-effort clean reconstruction of the script's evident intent
# (detect OS/arch, resolve the latest GitHub release tag, download and
# extract the matching archive, install the `saka` binary) — verify it
# against the actual release layout before using it.
set -eu

REPO="sirerun/saka"
VERSION="${SAKA_VERSION:-latest}"

os=$(uname -s)
arch=$(uname -m)
case "$os" in
  Darwin) os="darwin" ;;
  Linux)  os="linux" ;;
  *) echo "unsupported OS: $os" >&2; exit 1 ;;
esac
case "$arch" in
  x86_64|amd64) arch="amd64" ;;
  arm64|aarch64) arch="arm64" ;;
  *) echo "unsupported arch: $arch" >&2; exit 1 ;;
esac

if [ "$VERSION" = "latest" ]; then
  VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
    | grep -o '"tag_name": *"[^"]*"' | cut -d'"' -f4)
fi

ext="tar.gz"
if [ "$os" = "windows" ]; then ext="zip"; fi
name="saka_${VERSION#v}_${os}_${arch}"
url="https://github.com/${REPO}/releases/download/${VERSION}/${name}.${ext}"

echo "Installing saka ${VERSION} (${os}/${arch})..."

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

curl -fsSL "$url" | tar -xz -C "$tmp" saka

dest="${SAKA_INSTALL_DIR:-/usr/local/bin}"
mkdir -p "$dest" 2>/dev/null || dest="$HOME/.local/bin" && mkdir -p "$dest"
install "$tmp/saka" "$dest/saka"

echo "✓ saka installed to $dest/saka"
echo "  Try: saka search \"open source LLMs\" --markdown"
