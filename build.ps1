# PowerShell ビルドスクリプト
# Windows環境でMakefileの代わりに使用できます

param(
    [Parameter(Position=0)]
    [ValidateSet("build", "run", "test", "test-coverage", "deps", "clean", "help")]
    [string]$Command = "help"
)

$BinaryName = "gem-server"
$CmdPath = "cmd/main.go"
$BuildDir = "bin"
$CoverageDir = "coverage"

function Build {
    Write-Host "Building $BinaryName..." -ForegroundColor Green
    if (-not (Test-Path $BuildDir)) {
        New-Item -ItemType Directory -Path $BuildDir | Out-Null
    }
    go build -o "$BuildDir/$BinaryName.exe" $CmdPath
    if ($LASTEXITCODE -eq 0) {
        Write-Host "Build complete: $BuildDir/$BinaryName.exe" -ForegroundColor Green
    } else {
        Write-Host "Build failed" -ForegroundColor Red
        exit 1
    }
}

function Run {
    Write-Host "Running $BinaryName..." -ForegroundColor Green
    go run $CmdPath
}

function Test {
    Write-Host "Running tests..." -ForegroundColor Green
    go test -v ./...
}

function TestCoverage {
    Write-Host "Running tests with coverage..." -ForegroundColor Green
    if (-not (Test-Path $CoverageDir)) {
        New-Item -ItemType Directory -Path $CoverageDir | Out-Null
    }
    go test -coverprofile="$CoverageDir/coverage.out" ./...
    go tool cover -html="$CoverageDir/coverage.out" -o "$CoverageDir/coverage.html"
    Write-Host "Coverage report generated: $CoverageDir/coverage.html" -ForegroundColor Green
}

function Deps {
    Write-Host "Installing dependencies..." -ForegroundColor Green
    go mod download
    go mod tidy
}

function Clean {
    Write-Host "Cleaning..." -ForegroundColor Green
    if (Test-Path $BuildDir) {
        Remove-Item -Recurse -Force $BuildDir
    }
    if (Test-Path $CoverageDir) {
        Remove-Item -Recurse -Force $CoverageDir
    }
    go clean
}

function Help {
    Write-Host "Available commands:" -ForegroundColor Cyan
    Write-Host "  build          - Build the application"
    Write-Host "  run            - Run the application"
    Write-Host "  test           - Run tests"
    Write-Host "  test-coverage  - Run tests with coverage report"
    Write-Host "  deps           - Install dependencies"
    Write-Host "  clean          - Clean build artifacts"
    Write-Host "  help           - Show this help message"
    Write-Host ""
    Write-Host "Usage: .\build.ps1 <command>" -ForegroundColor Yellow
}

switch ($Command) {
    "build" { Build }
    "run" { Run }
    "test" { Test }
    "test-coverage" { TestCoverage }
    "deps" { Deps }
    "clean" { Clean }
    "help" { Help }
    default { Help }
}
