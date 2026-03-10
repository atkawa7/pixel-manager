$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$RootDir = Split-Path -Parent $ScriptDir
$FrontendDir = Join-Path $RootDir "frontend"
$OutDir = Join-Path $RootDir "bin"
$OutBin = Join-Path $OutDir "pixel-manager.exe"

if (-not (Get-Command npm -ErrorAction SilentlyContinue)) {
  throw "npm is required but was not found in PATH."
}

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
  throw "go is required but was not found in PATH."
}

Write-Host "==> Building frontend into internal/httpserver/public"
Push-Location $FrontendDir
try {
  $NodeModules = Join-Path $FrontendDir "node_modules"
  if (-not (Test-Path $NodeModules)) {
    npm ci
  }
  npm run build
}
finally {
  Pop-Location
}

$EmbeddedPublicDir = Join-Path $RootDir "internal/httpserver/public"
$EmbeddedIndex = Join-Path $EmbeddedPublicDir "index.html"
$EmbeddedAssetsDir = Join-Path $EmbeddedPublicDir "assets"
$HasJs = @(Get-ChildItem -Path $EmbeddedAssetsDir -Filter *.js -ErrorAction SilentlyContinue).Count -gt 0
$HasCss = @(Get-ChildItem -Path $EmbeddedAssetsDir -Filter *.css -ErrorAction SilentlyContinue).Count -gt 0

if (-not (Test-Path $EmbeddedIndex) -or -not (Test-Path $EmbeddedAssetsDir) -or -not $HasJs -or -not $HasCss) {
  throw "Embedded frontend assets missing. Expected index.html and assets/*.js + assets/*.css in internal/httpserver/public."
}

Write-Host "==> Building Go binary"
New-Item -ItemType Directory -Force -Path $OutDir | Out-Null
Push-Location $RootDir
try {
  go build -a -o $OutBin ./cmd/pixel-manager
}
finally {
  Pop-Location
}

Write-Host "==> Done"
Write-Host "Binary: $OutBin"
