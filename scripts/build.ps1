[CmdletBinding()]
param(
    [string]$Version = '0.4.9',
    [string]$BinaryName = 'wisp.exe',
    [string]$Go = 'go',
    [string]$MpvDirectory = '',
    [string]$ReleaseBaseUrl = ''
)
$ErrorActionPreference = 'Stop'
if ($Version -notmatch '^[0-9A-Za-z.-]+$') { throw 'Version may contain letters, numbers, dots and hyphens only.' }
if ($BinaryName -notmatch '^[A-Za-z0-9.-]+\.exe$') { throw 'BinaryName must be a simple .exe filename.' }
$root = Split-Path -Parent $PSScriptRoot
$dist = Join-Path $root 'dist'
$stage = Join-Path $dist ('package-' + [Guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $stage -Force | Out-Null
Push-Location $root
try {
    & $Go test ./...
    if ($LASTEXITCODE -ne 0) { throw 'Tests failed.' }
    & $Go vet ./...
    if ($LASTEXITCODE -ne 0) { throw 'Go vet failed.' }
    & $Go build -trimpath -ldflags "-s -w -X main.version=$Version" -o (Join-Path $stage 'wisp.exe') ./cmd/wisp
    if ($LASTEXITCODE -ne 0) { throw 'Build failed.' }
    Copy-Item -LiteralPath (Join-Path $stage 'wisp.exe') -Destination (Join-Path $dist $BinaryName) -Force
    foreach ($document in @('README.md','README.ru.md','LICENSE','THIRD_PARTY.md')) {
        Copy-Item -LiteralPath (Join-Path $root $document) -Destination $stage
    }
    Copy-Item -LiteralPath (Join-Path $PSScriptRoot 'setup-audio.ps1') -Destination $stage
    $packageDocs = Join-Path $stage 'docs'
    New-Item -ItemType Directory -Path $packageDocs -Force | Out-Null
    Get-ChildItem -LiteralPath (Join-Path $root 'docs') -File -Filter '*.md' |
        Where-Object Name -ne 'CUE_RESEARCH.md' | Copy-Item -Destination $packageDocs
    Copy-Item -LiteralPath (Join-Path $root 'docs/assets') -Destination $packageDocs -Recurse
    $notices = Join-Path $stage 'licenses'
    New-Item -ItemType Directory -Path $notices -Force | Out-Null
    $goRoot = & $Go env GOROOT
    if ($LASTEXITCODE -ne 0) { throw 'Could not locate Go runtime notices.' }
    $goNotices = Join-Path $notices 'go-runtime'
    New-Item -ItemType Directory -Path $goNotices -Force | Out-Null
    Copy-Item -LiteralPath (Join-Path $goRoot 'LICENSE') -Destination $goNotices
    if (Test-Path -LiteralPath (Join-Path $goRoot 'PATENTS')) {
        Copy-Item -LiteralPath (Join-Path $goRoot 'PATENTS') -Destination $goNotices
    }
    $modules = & $Go list -m -f '{{.Path}}|{{.Version}}|{{.Dir}}' all
    if ($LASTEXITCODE -ne 0) { throw 'Could not enumerate dependency notices.' }
    foreach ($module in $modules) {
        $parts = $module -split '\|',3
        if ($parts.Length -ne 3 -or !$parts[1] -or !$parts[2]) { continue }
        $name = ($parts[0] + '@' + $parts[1]) -replace '[/\\:]','_'
        $target = Join-Path $notices $name
        $files = Get-ChildItem -LiteralPath $parts[2] -File | Where-Object { $_.Name -match '^(LICENSE|LICENCE|COPYING|NOTICE)' }
        if ($files) { New-Item -ItemType Directory -Path $target -Force | Out-Null; $files | Copy-Item -Destination $target }
    }
    if ($MpvDirectory) {
        if (!(Test-Path -LiteralPath (Join-Path $MpvDirectory 'mpv.exe'))) { throw 'MpvDirectory must contain mpv.exe and its distribution notices.' }
        Copy-Item -Path (Join-Path $MpvDirectory '*') -Destination $stage -Recurse -Force
    }
    $archive = Join-Path $dist "wisp-$Version-windows-amd64.zip"
    Compress-Archive -Path (Join-Path $stage '*') -DestinationPath $archive -Force
    $hash = (Get-FileHash -LiteralPath $archive -Algorithm SHA256).Hash.ToLowerInvariant()
    $installer = Get-Content -LiteralPath (Join-Path $PSScriptRoot 'install.ps1') -Raw
    if ($ReleaseBaseUrl) {
        $uri = [Uri]$ReleaseBaseUrl
        if ($uri.Scheme -ne 'https' -or $ReleaseBaseUrl.Contains("'")) { throw 'ReleaseBaseUrl must be an HTTPS URL without quotes.' }
        $url = $ReleaseBaseUrl.TrimEnd('/') + '/' + [IO.Path]::GetFileName($archive)
        $installer = $installer.Replace("[string]`$ArchiveUrl = ''", "[string]`$ArchiveUrl = '$url'")
        $installer = $installer.Replace("[string]`$Sha256 = ''", "[string]`$Sha256 = '$hash'")
    }
    # GitHub serves downloads as octet-stream; PowerShell 5.1 irm treats a BOM
    # as literal characters. Keep the pipe-to-iex bootstrap ASCII and BOM-free.
    if ($installer -match '[^\x00-\x7F]') { throw 'The pipe installer must contain ASCII text only.' }
    [IO.File]::WriteAllText((Join-Path $dist 'install.ps1'),$installer,[Text.UTF8Encoding]::new($false))
    "$hash  $([IO.Path]::GetFileName($archive))" | Set-Content -LiteralPath (Join-Path $dist 'SHA256SUMS') -Encoding ascii
    [PSCustomObject]@{version=$Version;archive=[IO.Path]::GetFileName($archive);sha256=$hash;mpvIncluded=[bool]$MpvDirectory} | ConvertTo-Json | Set-Content -LiteralPath (Join-Path $dist 'release.json') -Encoding utf8
    Write-Host "Built $archive"
} finally {
    Pop-Location
    $resolvedStage = [IO.Path]::GetFullPath($stage)
    $resolvedDist = [IO.Path]::GetFullPath($dist).TrimEnd('\') + '\'
    if ($resolvedStage.StartsWith($resolvedDist,[StringComparison]::OrdinalIgnoreCase)) { Remove-Item -LiteralPath $resolvedStage -Recurse -Force }
}
