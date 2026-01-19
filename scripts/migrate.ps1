# マイグレーション実行スクリプト（PowerShell版）

param(
    [Parameter(Position=0)]
    [ValidateSet("up", "down", "create", "version", "force")]
    [string]$Command = "up",
    
    [Parameter(Position=1)]
    [int]$Version = 0,
    
    [string]$Name = ""
)

$MigrationsDir = "migrations"
$DatabaseURL = $env:DATABASE_URL

if (-not $DatabaseURL) {
    # 環境変数から設定を読み込む
    $env:ENVIRONMENT = if ($env:ENVIRONMENT) { $env:ENVIRONMENT } else { "development" }
    
    # .envファイルを読み込む
    if (Test-Path ".env.$env:ENVIRONMENT") {
        Get-Content ".env.$env:ENVIRONMENT" | ForEach-Object {
            if ($_ -match '^\s*([^#][^=]+)=(.*)$') {
                $key = $matches[1].Trim()
                $value = $matches[2].Trim()
                [Environment]::SetEnvironmentVariable($key, $value, "Process")
            }
        }
    } elseif (Test-Path ".env") {
        Get-Content ".env" | ForEach-Object {
            if ($_ -match '^\s*([^#][^=]+)=(.*)$') {
                $key = $matches[1].Trim()
                $value = $matches[2].Trim()
                [Environment]::SetEnvironmentVariable($key, $value, "Process")
            }
        }
    }
    
    $DBHost = $env:DB_HOST
    $DBPort = $env:DB_PORT
    $DBUser = $env:DB_USER
    $DBPassword = $env:DB_PASSWORD
    $DBName = $env:DB_NAME
    
    if (-not $DBHost -or -not $DBName) {
        Write-Host "Error: Database configuration not found. Please set DATABASE_URL or .env file." -ForegroundColor Red
        exit 1
    }
    
    $DatabaseURL = "mysql://${DBUser}:${DBPassword}@tcp(${DBHost}:${DBPort})/${DBName}?multiStatements=true"
}

$MigratePath = "migrate"
if (-not (Get-Command $MigratePath -ErrorAction SilentlyContinue)) {
    Write-Host "Error: migrate command not found. Please install golang-migrate:" -ForegroundColor Red
    Write-Host "  go install -tags 'mysql' github.com/golang-migrate/migrate/v4/cmd/migrate@latest" -ForegroundColor Yellow
    exit 1
}

switch ($Command) {
    "up" {
        Write-Host "Running migrations up..." -ForegroundColor Green
        & $MigratePath -path $MigrationsDir -database $DatabaseURL up
    }
    "down" {
        if ($Version -gt 0) {
            Write-Host "Running migrations down to version $Version..." -ForegroundColor Green
            & $MigratePath -path $MigrationsDir -database $DatabaseURL down $Version
        } else {
            Write-Host "Running migrations down 1 step..." -ForegroundColor Green
            & $MigratePath -path $MigrationsDir -database $DatabaseURL down 1
        }
    }
    "create" {
        if (-not $Name) {
            Write-Host "Error: Name is required for create command" -ForegroundColor Red
            Write-Host "Usage: .\scripts\migrate.ps1 create <name>" -ForegroundColor Yellow
            exit 1
        }
        Write-Host "Creating migration: $Name..." -ForegroundColor Green
        & $MigratePath create -ext sql -dir $MigrationsDir -seq $Name
    }
    "version" {
        Write-Host "Current migration version:" -ForegroundColor Green
        & $MigratePath -path $MigrationsDir -database $DatabaseURL version
    }
    "force" {
        if ($Version -eq 0) {
            Write-Host "Error: Version is required for force command" -ForegroundColor Red
            Write-Host "Usage: .\scripts\migrate.ps1 force <version>" -ForegroundColor Yellow
            exit 1
        }
        Write-Host "Forcing migration version to $Version..." -ForegroundColor Yellow
        & $MigratePath -path $MigrationsDir -database $DatabaseURL force $Version
    }
}

if ($LASTEXITCODE -ne 0) {
    Write-Host "Migration failed with exit code $LASTEXITCODE" -ForegroundColor Red
    exit $LASTEXITCODE
}
