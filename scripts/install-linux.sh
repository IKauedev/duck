#!/usr/bin/env sh
set -eu

# Instala o Duck no Linux/macOS sem compilar.
# Uso: ./scripts/install-linux.sh [pasta_destino]

REPO="IKauedev/duck"
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Arquitetura nao suportada: $ARCH" >&2; exit 1 ;;
esac

case "$OS" in
  linux) ;;
  darwin) OS="darwin" ;;
  *) echo "Sistema nao suportado: $OS" >&2; exit 1 ;;
esac

INSTALL_DIR="${1:-$HOME/bin}"
WORKDIR="${TMPDIR:-/tmp}/duck-install-$$"
ARCHIVE="$WORKDIR/duck_${OS}_${ARCH}.tar.gz"
URL="https://github.com/$REPO/releases/latest/download/duck_${OS}_${ARCH}.tar.gz"

mkdir -p "$WORKDIR" "$INSTALL_DIR"

echo "Duck - instalacao via shell"
echo "Destino: $INSTALL_DIR"
echo "URL: $URL"

if command -v curl >/dev/null 2>&1; then
  curl -fsSL -o "$ARCHIVE" "$URL"
elif command -v wget >/dev/null 2>&1; then
  wget -qO "$ARCHIVE" "$URL"
else
  echo "Erro: instale curl ou wget" >&2
  exit 1
fi

tar -xzf "$ARCHIVE" -C "$WORKDIR"
install -m 0755 "$WORKDIR/duck" "$INSTALL_DIR/duck"

"$INSTALL_DIR/duck" install --dir "$INSTALL_DIR" --force

echo
echo "Duck instalado com sucesso."
echo "Abra um novo terminal e execute: duck help"
