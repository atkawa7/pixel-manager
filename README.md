# Pixel Manager

Pixel Manager is a Go-based control plane for Unreal Engine Pixel Streaming instances.
It manages instance lifecycle, model configuration, cluster manager discovery, and provides a web UI for operations.

## Overview

- Backend: Go HTTP server (`cmd/pixel-manager`)
- Frontend: React + TypeScript + MUI SPA (`frontend/`)
- State: etcd
- Production UI delivery: embedded static assets in the Go binary (`internal/httpserver/public`)

## Features

- Start, stop, and inspect Pixel Streaming instances
- Cluster-aware manager discovery and remote delegation
- Model registry management
- Instance log capture and retrieval
- Runtime config visibility in UI (Settings page)
- Advanced launch options in UI:
  - Codec (`H264`, `VP8`, `VP9`, `AV1`)
  - Resolution presets (`ResX`, `ResY`)
  - Quality (`PixelStreamingEncoderMinQuality`, `PixelStreamingEncoderMaxQuality`)
  - WebRTC bitrate (Mbps input, converted to bps)
  - HUD stats, audio receive/transmit toggles
  - D3D renderer (`d3d11`/`d3d12`) and `d3ddebug`

## Configuration

The backend supports defaults + YAML + environment overrides.

Precedence:
1. Built-in defaults
2. YAML file (`CONFIG_FILE`, `CONFIG_PATH`, else `config.yaml`)
3. Environment variables (highest priority)

Example `config.yaml`:

```yaml
etcd:
  host: localhost
  port: 2379
  user: root
  password: password
manager_port: 4000
max_instances: 3
signal_server_url: http://127.0.0.1
```

## Running Locally

### 1. Start dependencies

You need a reachable etcd instance.

If using Docker Compose:

```bash
docker compose up -d
```

### 2. Run backend

```bash
go run ./cmd/pixel-manager
```

Backend default URL:

```text
http://localhost:4000
```

### 3. Run frontend in dev mode (optional)

```bash
cd frontend
npm install
npm run dev
```

In dev mode, frontend API base defaults to `http://localhost:4000`.

## Production Build (Embedded UI)

Build frontend assets into `internal/httpserver/public`, then compile Go binary.

### macOS / Linux (Bash)

```bash
chmod +x scripts/build-prod.sh
./scripts/build-prod.sh
```

### Windows (PowerShell)

```powershell
.\scripts\build-prod.ps1
```

Run the produced executable:

```powershell
.\bin\pixel-manager.exe
```

### Manual

```bash
cd frontend
npm install
npm run build

cd ..
go build ./cmd/pixel-manager
```

The resulting binary serves the SPA directly.
In production build, frontend API base defaults to `/api`.

Quick verification after build:

```bash
find internal/httpserver/public -maxdepth 3 -type f
```

Windows PowerShell equivalent:

```powershell
Get-ChildItem -Recurse internal/httpserver/public
```

## API Summary

Core endpoints (also available under `/api/...`):

- `GET /instances`
- `POST /instances`
- `DELETE /instances`
- `GET /instances/{id}`
- `DELETE /instances/{id}`
- `GET /instances/{id}/logs?tail=300`
- `GET /models`
- `POST /models`
- `DELETE /models/{name}`
- `GET /managers`
- `GET /config`

## Logs

Process logs are stored on disk per instance:

```text
logs/<instance_id>/instance.log
```

The UI reads logs via `GET /instances/{id}/logs`.

## UI Routes

- `/portal`
- `/managers`
- `/models`
- `/settings`

## Notes

- Sensitive config values (for example etcd password) are redacted in `/config` responses and startup config logs.
- Use `GOCACHE=/tmp/go-build go build ./...` if your local Go build cache path is permission restricted.
