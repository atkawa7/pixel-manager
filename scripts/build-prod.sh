#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
FRONTEND_DIR="${ROOT_DIR}/frontend"
OUT_DIR="${ROOT_DIR}/bin"
OUT_BIN="${OUT_DIR}/pixel-manager"

if ! command -v npm >/dev/null 2>&1; then
  echo "npm is required but not found in PATH." >&2
  exit 1
fi

if ! command -v go >/dev/null 2>&1; then
  echo "go is required but not found in PATH." >&2
  exit 1
fi

echo "==> Building frontend into internal/httpserver/public"
cd "${FRONTEND_DIR}"
if [[ ! -d node_modules ]]; then
  npm ci
fi
npm run build

echo "==> Building Go binary"
cd "${ROOT_DIR}"
mkdir -p "${OUT_DIR}"
go build -o "${OUT_BIN}" ./cmd/pixel-manager

echo "==> Done"
echo "Binary: ${OUT_BIN}"

