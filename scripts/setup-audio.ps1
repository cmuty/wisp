[CmdletBinding()]
param(
    [string]$ToolsDir = '',
    [string]$MpvDirectory = '',
    [string]$YtdlpPath = '',
    [switch]$WithoutMpv
)
$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'
# Packaged scripts sit beside wisp.exe; source scripts sit under scripts/.
if (!$ToolsDir -or !$MpvDirectory) {
    $audioRoot = if (Test-Path -LiteralPath (Join-Path $PSScriptRoot 'wisp.exe')) { $PSScriptRoot } else { Join-Path (Split-Path -Parent $PSScriptRoot) 'dist' }
    if (!$ToolsDir) { $ToolsDir = Join-Path $audioRoot 'tools' }
    if (!$MpvDirectory) { $MpvDirectory = Join-Path $audioRoot 'mpv' }
}
$scratch = Join-Path ([IO.Path]::GetTempPath()) ('wisp-audio-' + [Guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $scratch,$ToolsDir -Force | Out-Null
function Get-CheckedFile([string]$Url,[string]$Hash,[string]$Path) {
    Invoke-WebRequest -UseBasicParsing -Uri $Url -OutFile $Path
    if ((Get-FileHash -LiteralPath $Path -Algorithm SHA256).Hash -ne $Hash) { throw 'Audio dependency checksum mismatch.' }
}
try {
    $ytHash = '66674953fe251b89f4d08c5f0e35e0728679bd67ab3d7d05c0562af101dd3e7a'
    $ytTarget = Join-Path $ToolsDir 'yt-dlp.exe'
    if (!(Test-Path -LiteralPath $ytTarget) -or (Get-FileHash -LiteralPath $ytTarget -Algorithm SHA256).Hash -ne $ytHash) {
        $ytTemp = Join-Path $scratch 'yt-dlp.exe'
        if ($YtdlpPath) {
            if ((Get-FileHash -LiteralPath $YtdlpPath -Algorithm SHA256).Hash -ne $ytHash) { throw 'yt-dlp checksum mismatch.' }
            Copy-Item -LiteralPath $YtdlpPath -Destination $ytTemp
        } else {
            Get-CheckedFile 'https://github.com/yt-dlp/yt-dlp/releases/download/2026.08.19/yt-dlp.exe' $ytHash $ytTemp
        }
        Copy-Item -LiteralPath $ytTemp -Destination $ytTarget -Force
    }
    if (!(Get-Command node -ErrorAction SilentlyContinue) -and !(Get-Command deno -ErrorAction SilentlyContinue) -and !(Test-Path -LiteralPath (Join-Path $ToolsDir 'deno.exe'))) {
        $denoZip = Join-Path $scratch 'deno.zip'
        Get-CheckedFile 'https://github.com/denoland/deno/releases/download/v2.9.7/deno-x86_64-pc-windows-msvc.zip' 'a0c3101b4158d1dfb7d6a78a7bf0f3de80c96bb423c152beec8beb22786f2238' $denoZip
        Expand-Archive -LiteralPath $denoZip -DestinationPath (Join-Path $scratch 'deno')
        Copy-Item -LiteralPath (Join-Path $scratch 'deno\deno.exe') -Destination $ToolsDir -Force
    }
    if (!$WithoutMpv -and !(Get-Command mpv -ErrorAction SilentlyContinue) -and !(Test-Path -LiteralPath (Join-Path $MpvDirectory 'mpv.exe')) -and !($env:WISP_MPV -and (Test-Path -LiteralPath $env:WISP_MPV))) {
        $mpvZip = Join-Path $scratch 'mpv.zip'
        Get-CheckedFile 'https://github.com/mpv-player/mpv/releases/download/git-release/mpv-v0.41.0-dev-ge471496c5-39465194068-x86_64-w64-mingw32-full.zip' '9fc5bcabb671a8878ef32fc4b3c3736f11605e25ed367af49d35b2aa43a8133b' $mpvZip
        Expand-Archive -LiteralPath $mpvZip -DestinationPath $MpvDirectory -Force
    }
    [PSCustomObject]@{ytDlp='2026.08.19';sha256=$ytHash;source='https://github.com/yt-dlp/yt-dlp/releases/tag/2026.08.19'} | ConvertTo-Json | Set-Content -LiteralPath (Join-Path $ToolsDir 'manifest.json') -Encoding utf8
    Write-Host 'Audio tools are ready. No account token is needed for public sources.'
} finally {
    $resolvedScratch = [IO.Path]::GetFullPath($scratch)
    $resolvedTemp = [IO.Path]::GetFullPath([IO.Path]::GetTempPath()).TrimEnd('\') + '\'
    if ($resolvedScratch.StartsWith($resolvedTemp,[StringComparison]::OrdinalIgnoreCase)) { Remove-Item -LiteralPath $resolvedScratch -Recurse -Force }
}
