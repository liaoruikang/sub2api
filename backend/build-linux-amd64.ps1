$ErrorActionPreference = 'Stop'

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
Push-Location $scriptDir

$previousGOOS = $env:GOOS
$previousGOARCH = $env:GOARCH
$previousCGO_ENABLED = $env:CGO_ENABLED

try {
    $version = (Get-Content .\cmd\server\VERSION -Raw).Trim()
    $outputPath = Join-Path $scriptDir 'sub2api'

    $env:GOOS = 'linux'
    $env:GOARCH = 'amd64'
    $env:CGO_ENABLED = '0'

    if (Test-Path -LiteralPath $outputPath) {
        Remove-Item -LiteralPath $outputPath -Force
    }

    go build -tags embed -gcflags=all=-d=ssa/nilcheckelim/off -o sub2api ./cmd/server

    Write-Host "Built .\sub2api for linux/amd64"
} finally {
    if ($null -eq $previousGOOS) { Remove-Item Env:GOOS -ErrorAction SilentlyContinue } else { $env:GOOS = $previousGOOS }
    if ($null -eq $previousGOARCH) { Remove-Item Env:GOARCH -ErrorAction SilentlyContinue } else { $env:GOARCH = $previousGOARCH }
    if ($null -eq $previousCGO_ENABLED) { Remove-Item Env:CGO_ENABLED -ErrorAction SilentlyContinue } else { $env:CGO_ENABLED = $previousCGO_ENABLED }

    Pop-Location
}
