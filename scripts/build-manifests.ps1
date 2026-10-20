[CmdletBinding()]
param(
    [Parameter(Mandatory)][string]$Version,
    [string]$Repository = 'cmuty/wisp'
)
$ErrorActionPreference = 'Stop'
if ($Version -notmatch '^\d+\.\d+\.\d+$' -or $Repository -notmatch '^[\w.-]+/[\w.-]+$') { throw 'Invalid package version or repository.' }
$root = Split-Path -Parent $PSScriptRoot
$dist = Join-Path $root 'dist'
$base = "https://github.com/$Repository/releases/download/v$Version"
$zipName = "wisp-$Version-windows-amd64.zip"
$exeName = "wisp-$Version-setup-x64.exe"
$zipHash = (Get-FileHash -LiteralPath (Join-Path $dist $zipName) -Algorithm SHA256).Hash.ToLowerInvariant()
$exeHash = (Get-FileHash -LiteralPath (Join-Path $dist $exeName) -Algorithm SHA256).Hash
$scoop = [ordered]@{
    version=$Version
    description='A little room for sound. A terminal music player for Windows.'
    homepage="https://github.com/$Repository"
    license='MIT'
    depends=@('extras/mpv','main/yt-dlp','main/deno')
    architecture=@{'64bit'=@{url="$base/$zipName";hash=$zipHash}}
    bin='wisp.exe'
    notes='Run wisp. Settings and collections are stored in %APPDATA%\wisp, independently of Scoop.'
    checkver=@{github="https://github.com/$Repository"}
    autoupdate=@{architecture=@{'64bit'=@{
        url=('https://github.com/'+$Repository+'/releases/download/v$version/wisp-$version-windows-amd64.zip')
        hash=@{url=('https://github.com/'+$Repository+'/releases/download/v$version/SHA256SUMS');regex='$sha256\s+wisp-$version-windows-amd64\.zip'}
    }}}
}
[IO.File]::WriteAllText((Join-Path $dist 'wisp.json'),($scoop | ConvertTo-Json -Depth 8),[Text.UTF8Encoding]::new($false))
$winget = Join-Path $dist "winget/manifests/c/cmuty/wisp/$Version"
New-Item -ItemType Directory -Path $winget -Force | Out-Null
$versionManifest = @"
# yaml-language-server: `$schema=https://aka.ms/winget-manifest.version.1.10.0.schema.json
PackageIdentifier: cmuty.wisp
PackageVersion: $Version
DefaultLocale: en-US
ManifestType: version
ManifestVersion: 1.10.0
"@
$localeManifest = @"
# yaml-language-server: `$schema=https://aka.ms/winget-manifest.defaultLocale.1.10.0.schema.json
PackageIdentifier: cmuty.wisp
PackageVersion: $Version
PackageLocale: en-US
Publisher: cmuty
PublisherUrl: https://github.com/cmuty
PublisherSupportUrl: https://github.com/$Repository/issues
PackageName: wisp
PackageUrl: https://github.com/$Repository
License: MIT
LicenseUrl: https://github.com/$Repository/blob/main/LICENSE
ShortDescription: A terminal music player with playlists, synchronized lyrics, and flowing themes.
Description: A Windows terminal music player with public-source search, local playlists, a persistent command line, synchronized lyrics, five themes, and English and Russian interfaces.
Moniker: wisp
Tags:
- music
- music-player
- terminal
- tui
ReleaseNotesUrl: https://github.com/$Repository/releases/tag/v$Version
ManifestType: defaultLocale
ManifestVersion: 1.10.0
"@
$installerManifest = @"
# yaml-language-server: `$schema=https://aka.ms/winget-manifest.installer.1.10.0.schema.json
PackageIdentifier: cmuty.wisp
PackageVersion: $Version
InstallerType: inno
Scope: user
MinimumOSVersion: 10.0.17763.0
InstallModes:
- interactive
- silent
- silentWithProgress
UpgradeBehavior: install
Commands:
- wisp
ReleaseDate: $([DateTime]::UtcNow.ToString('yyyy-MM-dd'))
Installers:
- Architecture: x64
  InstallerUrl: $base/$exeName
  InstallerSha256: $exeHash
  ProductCode: '{9ACD2363-C14C-4D5B-A38A-9FA4D819E5CF}_is1'
ManifestType: installer
ManifestVersion: 1.10.0
"@
foreach ($entry in @(@('cmuty.wisp.yaml',$versionManifest),@('cmuty.wisp.locale.en-US.yaml',$localeManifest),@('cmuty.wisp.installer.yaml',$installerManifest))) {
    [IO.File]::WriteAllText((Join-Path $winget $entry[0]),($entry[1]+"`n"),[Text.UTF8Encoding]::new($false))
}
Compress-Archive -LiteralPath (Join-Path $dist 'winget/manifests') -DestinationPath (Join-Path $dist 'winget-manifests.zip') -Force
Write-Output "Generated Scoop and WinGet manifests for $Version"
