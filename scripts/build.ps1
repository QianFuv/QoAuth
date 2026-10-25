$ErrorActionPreference = 'Stop'
$repository = Split-Path $PSScriptRoot -Parent
$priorEnvironment = @{}
foreach ($name in @('GOTOOLCHAIN', 'CGO_ENABLED', 'GOARCH', 'GOOS')) {
    $priorEnvironment[$name] = [Environment]::GetEnvironmentVariable($name, 'Process')
}
Push-Location $repository
try {
    Push-Location web
    try {
        npm ci
        if ($LASTEXITCODE -ne 0) { throw 'npm ci failed' }
        npm run build
        if ($LASTEXITCODE -ne 0) { throw 'Frontend build failed' }
    } finally { Pop-Location }
    $env:GOTOOLCHAIN = 'local'
    $env:CGO_ENABLED = '0'
    $env:GOARCH = 'amd64'
    foreach ($target in @('windows', 'linux')) {
        $env:GOOS = $target
        $extension = if ($target -eq 'windows') { '.exe' } else { '' }
        go build -trimpath -ldflags='-w' -o "dist/qoauth-$target-amd64$extension" ./cmd/qoauth
        if ($LASTEXITCODE -ne 0) { throw "$target build failed" }
    }
} finally {
    foreach ($name in $priorEnvironment.Keys) {
        [Environment]::SetEnvironmentVariable($name, $priorEnvironment[$name], 'Process')
    }
    Pop-Location
}
