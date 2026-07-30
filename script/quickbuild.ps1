# quickbuild.ps1 - Build and copy to target folder
# Usage: .\script\quickbuild.ps1

$ErrorActionPreference = "Stop"
$root = Split-Path $PSScriptRoot -Parent

$dllSrc = Join-Path $root "pst_bridge\dist\pst_bridge.dll"
$dllDst = Join-Path $root "internal\bridge\dll\pst_bridge.dll"

# Step 1: sync DLL
Write-Host "[1/3] Checking DLL..." -ForegroundColor Cyan
if (Test-Path $dllSrc) {
    $szSrc = (Get-Item $dllSrc).Length
    $szDst = if (Test-Path $dllDst) { (Get-Item $dllDst).Length } else { -1 }
    if ($szSrc -ne $szDst) {
        Copy-Item $dllSrc $dllDst -Force
        Write-Host "      DLL updated ($szSrc bytes)" -ForegroundColor Green
    } else {
        Write-Host "      DLL unchanged, skip" -ForegroundColor Gray
    }
} else {
    Write-Host "      DLL not found in dist, using existing embed" -ForegroundColor Yellow
}

# Step 2: build to temp ASCII path first (go build cannot handle non-ASCII output path)
$tmpExe = Join-Path $env:TEMP "pst_gm_build.exe"
Write-Host "[2/3] Building Go binary..." -ForegroundColor Cyan
Set-Location $root
& go build -trimpath -ldflags "-s -w -X main.version=GM-Local" -o $tmpExe .
if ($LASTEXITCODE -ne 0) {
    Write-Host "[FAIL] Build failed" -ForegroundColor Red
    exit 1
}

# Step 3: copy to final destination using .NET method (handles Chinese paths)
$destDir = "c:\Users\Administrator\Documents\trea\palworld-server-tool-main\参考\打包"
$destExe = [System.IO.Path]::Combine($destDir, "Pst魔改版-GM.exe")

[System.IO.File]::Copy($tmpExe, $destExe, $true)
Remove-Item $tmpExe -Force

$size = [math]::Round((Get-Item $destExe).Length / 1MB, 2)
Write-Host "[3/3] Output: $destExe" -ForegroundColor Cyan
Write-Host "      Size: ${size} MB" -ForegroundColor Green
Write-Host "Build complete!" -ForegroundColor Green
