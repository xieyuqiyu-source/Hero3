# Hero3 Dev - Start all services in one terminal (like macOS dev.sh)
$ErrorActionPreference = "Stop"
$ROOT = Split-Path -Parent $MyInvocation.MyCommand.Path

# Load .env
$envFile = Join-Path $ROOT "go\.env"
if (Test-Path $envFile) {
    Get-Content $envFile | ForEach-Object {
        if ($_ -match '^\s*([^#][^=]+)=(.*)$') {
            $key = $Matches[1].Trim()
            $val = $Matches[2].Trim().Trim('"')
            [Environment]::SetEnvironmentVariable($key, $val, "Process")
        }
    }
}

Write-Host ""
Write-Host "===== Hero3 Dev =====" -ForegroundColor Cyan
Write-Host ""

# SSH Tunnel
if ($env:HERO3_DB_TUNNEL_ENABLED -eq "true") {
    $localPort = if ($env:HERO3_DB_TUNNEL_LOCAL_PORT) { $env:HERO3_DB_TUNNEL_LOCAL_PORT } else { "3307" }
    $remotePort = if ($env:HERO3_DB_REMOTE_PORT) { $env:HERO3_DB_REMOTE_PORT } else { "3306" }
    $sshUser = if ($env:HERO3_DB_SSH_USER) { $env:HERO3_DB_SSH_USER } else { "root" }
    $sshHost = $env:HERO3_DB_SSH_HOST
    Write-Host "[tunnel] SSH $sshUser@$sshHost :$localPort -> :$remotePort" -ForegroundColor DarkGray
    $tunnel = Start-Process ssh -ArgumentList "-o","StrictHostKeyChecking=accept-new","-N","-L","${localPort}:127.0.0.1:${remotePort}","${sshUser}@${sshHost}" -PassThru -NoNewWindow
    Start-Sleep 2
}

# Start Go backend
Write-Host "[1/3] Go backend starting..." -ForegroundColor Green
$goProc = Start-Process -FilePath "go" -ArgumentList "run","./cmd/server" -WorkingDirectory (Join-Path $ROOT "go") -PassThru -NoNewWindow

Start-Sleep 1

# Start Web frontend
Write-Host "[2/3] Web frontend starting..." -ForegroundColor Green
$webProc = Start-Process -FilePath "cmd" -ArgumentList "/c","pnpm dev" -WorkingDirectory (Join-Path $ROOT "web") -PassThru -NoNewWindow

# Start Admin frontend
Write-Host "[3/3] Admin frontend starting..." -ForegroundColor Green
$adminProc = Start-Process -FilePath "cmd" -ArgumentList "/c","pnpm dev" -WorkingDirectory (Join-Path $ROOT "admin") -PassThru -NoNewWindow

Write-Host ""
Write-Host "===== Hero3 Dev Ready =====" -ForegroundColor Cyan
Write-Host "  Frontend: http://localhost:5173"
Write-Host "  Admin:    http://localhost:5174"
Write-Host "  Backend:  http://localhost:8080"
Write-Host ""
Write-Host "Press Ctrl+C to stop all services" -ForegroundColor Yellow
Write-Host ""

# Wait and cleanup on exit
try {
    while ($true) { Start-Sleep 1 }
} finally {
    Write-Host ""
    Write-Host "Stopping all services..." -ForegroundColor Yellow
    @($goProc, $webProc, $adminProc, $tunnel) | Where-Object { $_ -ne $null -and -not $_.HasExited } | ForEach-Object {
        Stop-Process -Id $_.Id -Force -ErrorAction SilentlyContinue
    }
    # Also kill child processes (node, go)
    Get-Process -Name "node" -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
    Write-Host "Done." -ForegroundColor Green
}
