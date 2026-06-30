$ErrorActionPreference = "Stop"

Set-Location (Split-Path -Parent $PSScriptRoot)

python -m pip install -e .[dev]
pyinstaller `
  --name OpenAIImageGenerator `
  --windowed `
  --clean `
  --noconfirm `
  --paths src `
  src/image_generator/main.py

Write-Host "Build complete: dist/OpenAIImageGenerator/OpenAIImageGenerator.exe"
