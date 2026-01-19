# gRPCコード生成スクリプト

$ErrorActionPreference = "Stop"

$ProtoDir = "internal\presentation\grpc\proto"
$OutputDir = "internal\presentation\grpc\pb"

Write-Host "Generating gRPC code from proto files..." -ForegroundColor Green

# 出力ディレクトリを作成
if (-not (Test-Path $OutputDir)) {
    New-Item -ItemType Directory -Path $OutputDir -Force | Out-Null
    Write-Host "Created directory: $OutputDir" -ForegroundColor Yellow
}

# protocコマンドを実行
$ProtoFile = Join-Path $ProtoDir "currency.proto"

if (-not (Test-Path $ProtoFile)) {
    Write-Host "Error: Proto file not found: $ProtoFile" -ForegroundColor Red
    exit 1
}

Write-Host "Generating code from: $ProtoFile" -ForegroundColor Cyan

# protocコマンドを実行
protoc `
    --go_out=$OutputDir `
    --go_opt=paths=source_relative `
    --go-grpc_out=$OutputDir `
    --go-grpc_opt=paths=source_relative `
    --proto_path=$ProtoDir `
    $ProtoFile

if ($LASTEXITCODE -ne 0) {
    Write-Host "Error: Failed to generate gRPC code" -ForegroundColor Red
    Write-Host "Make sure protoc and protoc-gen-go, protoc-gen-go-grpc are installed:" -ForegroundColor Yellow
    Write-Host "  go install google.golang.org/protobuf/cmd/protoc-gen-go@latest" -ForegroundColor Yellow
    Write-Host "  go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest" -ForegroundColor Yellow
    exit 1
}

Write-Host "gRPC code generated successfully!" -ForegroundColor Green
Write-Host "Output directory: $OutputDir" -ForegroundColor Cyan
