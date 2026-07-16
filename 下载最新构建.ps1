# 下载最新构建.ps1
# 从 GitHub Actions 下载最新的 Windows EXE 到本仓库目录
# 用法: 在 PowerShell 中执行，或右键"用 PowerShell 运行"
#
# 首次使用前，请设置环境变量 GH_TOKEN（推荐），或运行时手动输入 Token
# Token 需要 repo + actions:read 权限
# 设置环境变量方法（只需设置一次，永久生效）:
#   [System.Environment]::SetEnvironmentVariable("GH_TOKEN","你的token","User")

param(
    [string]$Token = $env:GH_TOKEN
)

$ErrorActionPreference = "Stop"

$REPO          = "bingyouxue/palworld-server-tool-gm"
$BRANCH        = "main"
$WORKFLOW_FILE = "dev-build.yml"
$ARTIFACT_NAME = "Pst-魔改-windows-x86_64"
$OUTPUT_DIR    = $PSScriptRoot

if (-not $Token) {
    $Token = Read-Host "请输入 GitHub Personal Access Token"
}

$headers = @{
    "Authorization"        = "Bearer $Token"
    "Accept"               = "application/vnd.github+json"
    "X-GitHub-Api-Version" = "2022-11-28"
}

Write-Host ""
Write-Host "正在查询最新构建..." -ForegroundColor Cyan

$runsUrl = "https://api.github.com/repos/$REPO/actions/workflows/$WORKFLOW_FILE/runs?branch=$BRANCH&status=success&per_page=5"
$runs    = Invoke-RestMethod -Uri $runsUrl -Headers $headers

if ($runs.workflow_runs.Count -eq 0) {
    Write-Host "没有找到成功的构建记录。请先推送代码到 main 分支触发构建。" -ForegroundColor Red
    Read-Host "按 Enter 退出"
    exit 1
}

$latestRun = $runs.workflow_runs[0]
$runId     = $latestRun.id
Write-Host "最新构建: Run #$runId  时间: $($latestRun.created_at)" -ForegroundColor Green

$artifactsUrl = "https://api.github.com/repos/$REPO/actions/runs/$runId/artifacts"
$artifacts    = Invoke-RestMethod -Uri $artifactsUrl -Headers $headers
$artifact     = $artifacts.artifacts | Where-Object { $_.name -eq $ARTIFACT_NAME } | Select-Object -First 1

if (-not $artifact) {
    Write-Host "未找到 artifact: $ARTIFACT_NAME" -ForegroundColor Red
    Read-Host "按 Enter 退出"
    exit 1
}

$sizeMB = [math]::Round($artifact.size_in_bytes / 1MB, 1)
Write-Host "找到 artifact: $($artifact.name)  大小: $sizeMB MB" -ForegroundColor Green

$tmpZip = Join-Path $env:TEMP "pst-build-$runId.zip"
Write-Host "正在下载..." -ForegroundColor Cyan
Invoke-WebRequest -Uri $artifact.archive_download_url -Headers $headers -OutFile $tmpZip

Write-Host "正在解压到: $OUTPUT_DIR" -ForegroundColor Cyan
Expand-Archive -Path $tmpZip -DestinationPath $OUTPUT_DIR -Force
Remove-Item $tmpZip

$exes = Get-ChildItem $OUTPUT_DIR -Filter "*.exe"
Write-Host ""
Write-Host "完成！解压出以下文件：" -ForegroundColor Green
foreach ($exe in $exes) {
    Write-Host "  $($exe.FullName)" -ForegroundColor Yellow
}

Write-Host ""
Read-Host "按 Enter 退出"
