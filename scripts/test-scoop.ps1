[CmdletBinding()]
param([Parameter(Mandatory)][string]$Version)
$ErrorActionPreference = 'Stop'
if (!$env:RUNNER_TEMP) { throw 'Run the Scoop smoke test on an isolated GitHub runner.' }
$root = Split-Path -Parent $PSScriptRoot
$env:SCOOP = Join-Path $env:RUNNER_TEMP 'wisp-scoop-smoke'
$checkout = Join-Path $env:SCOOP 'apps/scoop/current'
git clone --depth 1 https://github.com/ScoopInstaller/Scoop.git $checkout
if ($LASTEXITCODE -ne 0) { throw 'Could not obtain Scoop.' }
$scoop = Join-Path $checkout 'bin/scoop.ps1'
& $scoop bucket add main https://github.com/ScoopInstaller/Main
if ($LASTEXITCODE -ne 0) { throw 'Could not add the main bucket.' }
& $scoop bucket add extras https://github.com/ScoopInstaller/Extras
if ($LASTEXITCODE -ne 0) { throw 'Could not add the extras bucket.' }
$listener = [Net.Sockets.TcpListener]::new([Net.IPAddress]::Loopback,0)
$listener.Start()
$port = $listener.LocalEndpoint.Port
$listener.Stop()
$server = Start-Process -FilePath python -ArgumentList @('-m','http.server',"$port",'--bind','127.0.0.1','--directory',(Join-Path $root 'dist')) -WindowStyle Hidden -PassThru
try {
    for ($attempt=0; $attempt -lt 20; $attempt++) {
        try { Invoke-WebRequest "http://127.0.0.1:$port/" | Out-Null; break } catch { Start-Sleep -Milliseconds 250 }
    }
    $manifest = Get-Content (Join-Path $root 'dist/wisp.json') -Raw | ConvertFrom-Json
    $manifest.architecture.'64bit'.url = "http://127.0.0.1:$port/wisp-$Version-windows-amd64.zip"
    $localManifest = Join-Path $env:RUNNER_TEMP 'wisp.json'
    $manifest | ConvertTo-Json -Depth 8 | Set-Content -LiteralPath $localManifest -Encoding utf8
    & $scoop install $localManifest
    if ($LASTEXITCODE -ne 0) { throw 'Scoop installation failed.' }
    $launcher = Join-Path $env:SCOOP 'shims/wisp.exe'
    if ((& $launcher --version) -ne "wisp $Version") { throw 'Scoop installed the wrong build.' }
    & $scoop uninstall wisp
    if ($LASTEXITCODE -ne 0) { throw 'Scoop uninstall failed.' }
    Write-Output 'Scoop dependencies, archive hash, shim, version and uninstall passed.'
} finally {
    Stop-Process -Id $server.Id -Force -ErrorAction SilentlyContinue
}
