# Builds smeditor.exe on Windows with the Vue frontend embedded.
# Windows counterpart of `make build`; run from anywhere:
#   powershell -ExecutionPolicy Bypass -File .\build.ps1

$ErrorActionPreference = 'Stop'

$root = $PSScriptRoot
$frontend = Join-Path $root 'frontend'
$backend = Join-Path $root 'backend'
$distSrc = Join-Path $frontend 'dist'
$distDst = Join-Path $backend 'internal\webdist\dist'
$exe = Join-Path $root 'smeditor.exe'
$oldExe = Join-Path $root 'smeditor.exe.old'

function Invoke-Step {
    param([string]$Name, [scriptblock]$Command)
    Write-Host "==> $Name"
    & $Command
    if ($LASTEXITCODE -ne 0) {
        throw "$Name gagal (exit code $LASTEXITCODE)"
    }
}

Push-Location $frontend
try {
    # npm ci installs exactly what package-lock.json pins and never rewrites it.
    Invoke-Step 'npm ci' { npm ci }
    Invoke-Step 'npm run build' { npm run build }
} finally {
    Pop-Location
}

Write-Host "==> Salin frontend/dist ke backend/internal/webdist/dist"
if (Test-Path $distDst) {
    # Keep .gitkeep so the embed directory stays tracked in git.
    Get-ChildItem -Path $distDst -Force |
        Where-Object { $_.Name -ne '.gitkeep' } |
        Remove-Item -Recurse -Force
} else {
    New-Item -ItemType Directory -Path $distDst | Out-Null
}
Copy-Item -Path (Join-Path $distSrc '*') -Destination $distDst -Recurse -Force

# Windows refuses to overwrite a running .exe but does allow renaming it,
# so a running smeditor.exe is moved aside instead of failing the build.
# The old copy keeps running until it is closed.
if (Test-Path $oldExe) {
    try { Remove-Item -Force $oldExe } catch { Write-Host "smeditor.exe.old masih dipakai, dibiarkan." }
}
if (Test-Path $exe) {
    if (Test-Path $oldExe) {
        throw "smeditor.exe dan smeditor.exe.old sama-sama masih dipakai. Tutup smeditor.exe lalu jalankan ulang build.ps1."
    }
    Rename-Item -Path $exe -NewName 'smeditor.exe.old'
}

Push-Location $backend
try {
    $env:CGO_ENABLED = '0'
    Invoke-Step 'go build' { go build -tags embed_prod -o $exe ./cmd/smeditor }
} finally {
    Pop-Location
}

Write-Host "Selesai: $exe"
