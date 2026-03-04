# gint 构建脚本 (Windows PowerShell)

param(
    [Parameter(Position=0)]
    [string]$Command = "build"
)

$ErrorActionPreference = "Stop"

function Build-All {
    Write-Host "Building gint..." -ForegroundColor Green
    Push-Location gint
    go build .
    Pop-Location

    Write-Host "Building gintx..." -ForegroundColor Green
    Push-Location gintx
    go build ./...
    Pop-Location

    Write-Host "Building gint-gen..." -ForegroundColor Green
    Push-Location gint-gen
    go build -o bin/gint-gen.exe .
    Pop-Location

    Write-Host "Building registry..." -ForegroundColor Green
    Push-Location registry
    go build -o bin/registry.exe ./cmd/...
    Pop-Location

    Write-Host "Build complete!" -ForegroundColor Green
}

function Test-All {
    Write-Host "Running tests..." -ForegroundColor Green
    Push-Location gint
    go test -v ./...
    Pop-Location

    Push-Location gintx
    go test -v ./...
    Pop-Location

    Push-Location registry
    go test -v ./...
    Pop-Location
}

function Install-GintGen {
    Write-Host "Installing gint-gen to GOPATH/bin..." -ForegroundColor Green
    Push-Location gint-gen
    go install .
    Pop-Location
    Write-Host "Installed! Run: gint-gen --help" -ForegroundColor Green
}

function Run-Registry {
    Write-Host "Starting registry..." -ForegroundColor Green
    Push-Location registry
    go run ./cmd/...
    Pop-Location
}

function Run-Example {
    Write-Host "Starting example service..." -ForegroundColor Green
    Push-Location example/user
    go run ./cmd/main.go
    Pop-Location
}

function Clean {
    Write-Host "Cleaning..." -ForegroundColor Green
    Remove-Item -Recurse -Force gint-gen/bin -ErrorAction SilentlyContinue
    Remove-Item -Recurse -Force registry/bin -ErrorAction SilentlyContinue
    Write-Host "Clean complete!" -ForegroundColor Green
}

function Show-Help {
    Write-Host @"
Available commands:
  build       - Build all modules (default)
  test        - Run all tests
  install     - Install gint-gen to GOPATH/bin
  registry    - Run registry service
  example     - Run example service
  clean       - Clean build artifacts
  help        - Show this help

Usage:
  .\build.ps1 build
  .\build.ps1 install
  .\build.ps1 test
"@
}

# Main
switch ($Command) {
    "build"   { Build-All }
    "test"    { Test-All }
    "install" { Install-GintGen }
    "registry" { Run-Registry }
    "example" { Run-Example }
    "clean"   { Clean }
    "help"    { Show-Help }
    default   { Write-Host "Unknown command: $Command"; Show-Help }
}
