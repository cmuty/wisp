[CmdletBinding()]
param(
    [Parameter(Mandatory)][string]$Version,
    [string]$Compiler = 'C:\Program Files (x86)\Inno Setup 6\ISCC.exe'
)
$ErrorActionPreference = 'Stop'
if ($Version -notmatch '^\d+\.\d+\.\d+$') { throw 'Installer version must be major.minor.patch.' }
$root = Split-Path -Parent $PSScriptRoot
$dist = Join-Path $root 'dist'
$archive = Join-Path $dist "wisp-$Version-windows-amd64.zip"
if (!(Test-Path -LiteralPath $Compiler)) { throw 'Install Inno Setup 6 or pass -Compiler with the path to ISCC.exe.' }
$hash = (Get-FileHash -LiteralPath $archive -Algorithm SHA256).Hash.ToLowerInvariant()
& $Compiler "/DAppVersion=$Version" "/DPayloadDir=$dist" "/DReleaseId=$($hash.Substring(0,16))" (Join-Path $root 'packaging/windows/wisp.iss')
if ($LASTEXITCODE -ne 0) { throw 'Inno Setup compilation failed.' }
$exe = Join-Path $dist "wisp-$Version-setup-x64.exe"
$exeHash = (Get-FileHash -LiteralPath $exe -Algorithm SHA256).Hash.ToLowerInvariant()
"$exeHash  $([IO.Path]::GetFileName($exe))" | Add-Content -LiteralPath (Join-Path $dist 'SHA256SUMS') -Encoding ascii
Copy-Item -LiteralPath $exe -Destination (Join-Path $dist 'wisp-setup-x64.exe') -Force
"$exeHash  wisp-setup-x64.exe" | Add-Content -LiteralPath (Join-Path $dist 'SHA256SUMS') -Encoding ascii
$metadataPath = Join-Path $dist 'release.json'
$metadata = Get-Content -LiteralPath $metadataPath -Raw | ConvertFrom-Json
$metadata | Add-Member -NotePropertyName installer -NotePropertyValue ([IO.Path]::GetFileName($exe)) -Force
$metadata | Add-Member -NotePropertyName installerSha256 -NotePropertyValue $exeHash -Force
$metadata | Add-Member -NotePropertyName installerRequiresInternet -NotePropertyValue $true -Force
$metadata | ConvertTo-Json | Set-Content -LiteralPath $metadataPath -Encoding utf8
Write-Output "Built $exe"
