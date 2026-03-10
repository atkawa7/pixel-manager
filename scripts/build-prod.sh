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

EMBEDDED_PUBLIC_DIR="${ROOT_DIR}/internal/httpserver/public"
EMBEDDED_INDEX="${EMBEDDED_PUBLIC_DIR}/index.html"
EMBEDDED_ASSETS_DIR="${EMBEDDED_PUBLIC_DIR}/assets"

if [[ ! -f "${EMBEDDED_INDEX}" ]]; then
  echo "Missing embedded index file: ${EMBEDDED_INDEX}" >&2
  exit 1
fi
if [[ ! -d "${EMBEDDED_ASSETS_DIR}" ]]; then
  echo "Missing embedded assets directory: ${EMBEDDED_ASSETS_DIR}" >&2
  exit 1
fi
if ! ls "${EMBEDDED_ASSETS_DIR}"/*.js >/dev/null 2>&1; then
  echo "Missing embedded JS assets in ${EMBEDDED_ASSETS_DIR}" >&2
  exit 1
fi
if ! ls "${EMBEDDED_ASSETS_DIR}"/*.css >/dev/null 2>&1; then
  echo "Missing embedded CSS assets in ${EMBEDDED_ASSETS_DIR}" >&2
  exit 1
fi

echo "==> Building Go binary"
cd "${ROOT_DIR}"
mkdir -p "${OUT_DIR}"
go build -a -o "${OUT_BIN}" ./cmd/pixel-manager

echo "==> Done"
echo "Binary: ${OUT_BIN}"
