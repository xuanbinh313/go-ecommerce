param(
    [switch]$InstallDependencies
)

$ErrorActionPreference = "Stop"
$repoRoot = $PSScriptRoot

function ConvertTo-PowerShellLiteral([string]$Value) {
    return "'" + $Value.Replace("'", "''") + "'"
}

function Start-AppWindow([string]$Title, [string]$WorkingDirectory, [string]$Command) {
    $titleLiteral = ConvertTo-PowerShellLiteral $Title
    $directoryLiteral = ConvertTo-PowerShellLiteral $WorkingDirectory
    $childCommand = "`$Host.UI.RawUI.WindowTitle = $titleLiteral; Set-Location -LiteralPath $directoryLiteral; $Command"
    $encodedCommand = [Convert]::ToBase64String([Text.Encoding]::Unicode.GetBytes($childCommand))

    Start-Process -FilePath "powershell.exe" -ArgumentList @(
        "-NoExit",
        "-ExecutionPolicy", "Bypass",
        "-EncodedCommand", $encodedCommand
    )
}

if ($InstallDependencies) {
    foreach ($app in @("admin", "storefront")) {
        $appDirectory = Join-Path $repoRoot "www\$app"
        if (-not (Test-Path (Join-Path $appDirectory "node_modules"))) {
            Write-Host "Installing $app dependencies..."
            Push-Location $appDirectory
            try {
                pnpm install
            }
            finally {
                Pop-Location
            }
        }
    }
}

# The Go API does not load .env files itself, so expose its settings to the child process.
$envFile = Join-Path $repoRoot ".env"
if (Test-Path $envFile) {
    foreach ($line in Get-Content $envFile) {
        if ($line -match '^\s*([^#=\s]+)\s*=(.*)$') {
            [Environment]::SetEnvironmentVariable($Matches[1], $Matches[2].Trim(), "Process")
        }
    }
}

Start-AppWindow "Microservices API" $repoRoot "go run ./cmd/api"
Start-AppWindow "Admin app" (Join-Path $repoRoot "www\admin") "pnpm run dev"
Start-AppWindow "Storefront app" (Join-Path $repoRoot "www\storefront") "pnpm run dev"

Write-Host "Started API (http://localhost:8080), Admin (http://localhost:7001), and Storefront (http://localhost:8000)."
