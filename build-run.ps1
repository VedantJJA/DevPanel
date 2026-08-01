# =============================================================================
# DevPanel — Build & Run Script
# Usage: .\build-run.ps1 [OPTIONS]
#
# Options:
#   -Dev       Start frontend in dev mode (vite dev) instead of building
#   -BuildOnly Build both without starting servers
#   -SkipUI    Skip the UI build step (backend only)
#   -SkipBE    Skip the backend Go build (UI only)
#   -Port      Backend port (default: 8090)
# =============================================================================

param(
    [switch]$Dev,
    [switch]$BuildOnly,
    [switch]$SkipUI,
    [switch]$SkipBE,
    [int]$Port = 8090
)

$ErrorActionPreference = "Stop"
$Root     = $PSScriptRoot
$UIDir    = Join-Path $Root "ui"
$BinPath  = Join-Path $Root "devpnl.exe"

# ─── Colours ───────────────────────────────────────────────────────────────
function Info  ($msg) { Write-Host "  [INFO]  $msg" -ForegroundColor Cyan }
function OK    ($msg) { Write-Host "  [ OK ]  $msg" -ForegroundColor Green }
function Warn  ($msg) { Write-Host "  [WARN]  $msg" -ForegroundColor Yellow }
function Error ($msg) { Write-Host "  [ERR ]  $msg" -ForegroundColor Red; exit 1 }
function Step  ($msg) { Write-Host "`n━━━ $msg" -ForegroundColor Blue }

# ─── Pre-flight checks ─────────────────────────────────────────────────────
Step "Pre-flight checks"
if (-not (Get-Command "go" -ErrorAction SilentlyContinue))    { Error "Go is not installed or not in PATH" }
if (-not (Get-Command "node" -ErrorAction SilentlyContinue))  { Warn  "Node.js not found — UI build may fail" }
if (-not (Get-Command "npm" -ErrorAction SilentlyContinue))   { Warn  "npm not found — UI build may fail" }
OK "Environment looks good"

# ─── Build Backend ─────────────────────────────────────────────────────────
if (-not $SkipBE) {
    Step "Building Go backend (devpnl.exe)"
    Push-Location $Root
    try {
        & go build -o devpnl.exe ./cmd/server
        if ($LASTEXITCODE -ne 0) { Error "Go build failed" }
        OK "Backend built → $BinPath"
    } finally {
        Pop-Location
    }
}

# ─── Build / Dev Frontend ──────────────────────────────────────────────────
if (-not $SkipUI) {
    Step "Frontend: installing npm dependencies"
    Push-Location $UIDir
    try {
        if (-not (Test-Path "node_modules")) {
            Info "node_modules not found — running npm install"
            & npm install
            if ($LASTEXITCODE -ne 0) { Error "npm install failed" }
        } else {
            Info "node_modules present — skipping install"
        }

        if ($Dev) {
            OK "Dependencies ready (dev mode — will start vite dev server)"
        } else {
            Step "Building Svelte/SvelteKit frontend"
            & npm run build
            if ($LASTEXITCODE -ne 0) { Error "npm run build failed" }
            OK "Frontend built → ui/build/"
        }
    } finally {
        Pop-Location
    }
}

# ─── Exit if build-only ────────────────────────────────────────────────────
if ($BuildOnly) {
    OK "Build-only mode: both artifacts ready. Exiting."
    exit 0
}

# ─── Start servers ─────────────────────────────────────────────────────────
Step "Starting DevPanel servers"

$env:DEVPNL_PORT = $Port
$backendJob = $null
$frontendJob = $null

# Cleanup function
function Stop-All {
    if ($frontendJob) { Stop-Job -Job $frontendJob -ErrorAction SilentlyContinue; Remove-Job -Job $frontendJob -ErrorAction SilentlyContinue }
    if ($backendJob)  { Stop-Job -Job $backendJob  -ErrorAction SilentlyContinue; Remove-Job -Job $backendJob  -ErrorAction SilentlyContinue }
    Write-Host "`n  [STOP] All servers stopped." -ForegroundColor Magenta
}
Register-EngineEvent -SourceIdentifier PowerShell.Exiting -Action { Stop-All } | Out-Null

# Start backend
Info "Starting Go backend on port $Port..."
$backendJob = Start-Job -Name "devpnl-backend" -ScriptBlock {
    param($bin, $port)
    $env:DEVPNL_PORT = $port
    & $bin 2>&1
} -ArgumentList $BinPath, $Port

Start-Sleep -Milliseconds 1200

# Start frontend (dev or serve the static build)
if ($Dev) {
    Info "Starting Vite dev server (http://localhost:5173)..."
    $frontendJob = Start-Job -Name "devpnl-frontend" -ScriptBlock {
        param($dir)
        Set-Location $dir
        & npm run dev 2>&1
    } -ArgumentList $UIDir
    Start-Sleep -Milliseconds 800
    Info "Vite dev server proxies API to backend on port $Port"
    Write-Host ""
    OK "Frontend → http://localhost:5173"
    OK "Backend  → http://localhost:$Port"
} else {
    # Static build is served by the Go binary
    Write-Host ""
    OK "Frontend served by backend at http://localhost:$Port"
    OK "Backend API              at http://localhost:$Port/api"
}

Write-Host ""
Write-Host "  Press Ctrl+C to stop all servers." -ForegroundColor DarkGray
Write-Host ""

# ─── Tail both job outputs live ─────────────────────────────────────────────
try {
    while ($true) {
        # Print backend output
        $be = Receive-Job -Job $backendJob 2>&1
        foreach ($line in $be) { Write-Host "  [BE] $line" -ForegroundColor DarkCyan }

        # Print frontend output (if dev mode)
        if ($frontendJob) {
            $fe = Receive-Job -Job $frontendJob 2>&1
            foreach ($line in $fe) { Write-Host "  [FE] $line" -ForegroundColor DarkYellow }
        }

        # Check if backend died
        if ($backendJob.State -eq 'Failed' -or $backendJob.State -eq 'Completed') {
            Warn "Backend process exited unexpectedly."
            Receive-Job -Job $backendJob 2>&1 | ForEach-Object { Write-Host "  [BE] $_" -ForegroundColor Red }
            break
        }

        Start-Sleep -Milliseconds 400
    }
} finally {
    Stop-All
}
