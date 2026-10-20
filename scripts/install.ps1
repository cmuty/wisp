[CmdletBinding()]
param(
    [string]$ArchiveUrl = '',
    [string]$Sha256 = '',
    [string]$ArchivePath = '',
    [string]$MpvArchiveUrl = 'https://github.com/mpv-player/mpv/releases/download/git-release/mpv-v0.41.0-dev-ge471496c5-39465194068-x86_64-w64-mingw32-full.zip',
    [string]$MpvSha256 = '9fc5bcabb671a8878ef32fc4b3c3736f11605e25ed367af49d35b2aa43a8133b',
    [string]$MpvArchivePath = '',
    [string]$InstallDir = (Join-Path $env:LOCALAPPDATA 'Programs\wisp'),
    [switch]$NoPath,
    [switch]$WithoutMpv,
    [switch]$WithoutExtractor
)
$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'
if (![Environment]::Is64BitOperatingSystem) { throw 'This build requires 64-bit Windows.' }
$scratch = Join-Path ([IO.Path]::GetTempPath()) ('wisp-install-' + [Guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $scratch | Out-Null
try {
    if (!$ArchivePath) {
        if (!$ArchiveUrl -or $Sha256 -notmatch '^[a-fA-F0-9]{64}$') { throw 'Use a published installer with a pinned release, or pass -ArchivePath for a local build.' }
        if (([Uri]$ArchiveUrl).Scheme -ne 'https') { throw 'The release URL must use HTTPS.' }
        [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
        $ArchivePath = Join-Path $scratch 'release.zip'
        Invoke-WebRequest -UseBasicParsing -Uri $ArchiveUrl -OutFile $ArchivePath
    }
    $actual = (Get-FileHash -LiteralPath $ArchivePath -Algorithm SHA256).Hash.ToLowerInvariant()
    if ($Sha256 -and $actual -ne $Sha256.ToLowerInvariant()) { throw 'Release checksum mismatch. Nothing was installed.' }
    Add-Type -AssemblyName System.IO.Compression.FileSystem
    $zip = [IO.Compression.ZipFile]::OpenRead((Resolve-Path -LiteralPath $ArchivePath).Path)
    try {
        $size = 0L
        foreach ($entry in $zip.Entries) {
            $name = $entry.FullName.Replace('\','/')
            if ($name -match '(^/|:|(^|/)\.\.(/|$))') { throw 'Invalid path in release archive.' }
            $size += $entry.Length
            if ($size -gt 536870912) { throw 'Unpacked release exceeds 512 MiB.' }
        }
    } finally { $zip.Dispose() }
    $unpacked = Join-Path $scratch 'unpacked'
    Expand-Archive -LiteralPath $ArchivePath -DestinationPath $unpacked
    $binary = Join-Path $unpacked 'wisp.exe'
    if (!(Test-Path -LiteralPath $binary)) { throw 'Release archive does not contain wisp.exe.' }
    $hasMpv = (Test-Path -LiteralPath (Join-Path $unpacked 'mpv.exe')) -or (Test-Path -LiteralPath (Join-Path $unpacked 'mpv\mpv.exe')) -or [bool](Get-Command mpv -ErrorAction SilentlyContinue)
    if ($env:WISP_MPV -and (Test-Path -LiteralPath $env:WISP_MPV)) { $hasMpv = $true }
    if (!$WithoutMpv -and !$hasMpv) {
        Write-Host 'Installing the pinned mpv audio backend...'
        if (!$MpvArchivePath) {
            if (([Uri]$MpvArchiveUrl).Scheme -ne 'https') { throw 'The mpv URL must use HTTPS.' }
            $MpvArchivePath = Join-Path $scratch 'mpv.zip'
            Invoke-WebRequest -UseBasicParsing -Uri $MpvArchiveUrl -OutFile $MpvArchivePath
        }
        if ($MpvSha256 -notmatch '^[a-fA-F0-9]{64}$' -or (Get-FileHash -LiteralPath $MpvArchivePath -Algorithm SHA256).Hash -ne $MpvSha256) { throw 'mpv checksum mismatch. Nothing was installed.' }
        $backend = Join-Path $unpacked 'mpv'
        Expand-Archive -LiteralPath $MpvArchivePath -DestinationPath $backend
        if (!(Test-Path -LiteralPath (Join-Path $backend 'mpv.exe'))) { throw 'mpv archive does not contain mpv.exe.' }
    }
    $releaseDir = Join-Path $InstallDir ('releases\' + $actual.Substring(0,16))
	if (!$WithoutExtractor) {
		$audioSetup = Join-Path $unpacked 'setup-audio.ps1'
		if (Test-Path -LiteralPath $audioSetup) {
            # The helper is inside the verified archive. Evaluate it like this
            # bootstrap so default PowerShell file execution policy is unchanged.
            $setupBlock = [scriptblock]::Create([IO.File]::ReadAllText($audioSetup))
            & $setupBlock -ToolsDir (Join-Path $unpacked 'tools') -MpvDirectory (Join-Path $unpacked 'mpv') -WithoutMpv
        }
	}
    $bin = Join-Path $InstallDir 'bin'
    New-Item -ItemType Directory -Path $releaseDir,$bin -Force | Out-Null
    Copy-Item -Path (Join-Path $unpacked '*') -Destination $releaseDir -Recurse -Force
    $exe = Join-Path $releaseDir 'wisp.exe'
    $launcher = "@echo off`r`n`"%~dp0..\releases\$($actual.Substring(0,16))\wisp.exe`" %*`r`n"
    [IO.File]::WriteAllText((Join-Path $bin 'wisp.cmd'),$launcher,[Text.Encoding]::Default)
    if (!$NoPath) {
        $userPath = [Environment]::GetEnvironmentVariable('Path','User')
        $entries = @($userPath -split ';' | Where-Object { $_ })
        if ($entries -notcontains $bin) { [Environment]::SetEnvironmentVariable('Path', (($entries + $bin) -join ';'),'User') }
        if (($env:Path -split ';') -notcontains $bin) { $env:Path = $bin + ';' + $env:Path }
        Set-Alias -Name wisp -Value $exe -Scope Global
        if (!('WispEnvironment.Native' -as [type])) {
            Add-Type -TypeDefinition @'
using System;
using System.Runtime.InteropServices;
namespace WispEnvironment {
    public static class Native {
        [DllImport("user32.dll", CharSet = CharSet.Unicode, SetLastError = true)]
        public static extern IntPtr SendMessageTimeout(IntPtr hwnd, uint msg, UIntPtr wParam, string lParam, uint flags, uint timeout, out UIntPtr result);
    }
}
'@
        }
        $broadcastResult = [UIntPtr]::Zero
        [void][WispEnvironment.Native]::SendMessageTimeout([IntPtr]0xffff,0x001A,[UIntPtr]::Zero,'Environment',2,2000,[ref]$broadcastResult)
    }
    Write-Host "wisp installed. Run: wisp"
    if (!(Test-Path -LiteralPath (Join-Path $releaseDir 'mpv.exe')) -and !(Test-Path -LiteralPath (Join-Path $releaseDir 'mpv\mpv.exe')) -and !(Get-Command mpv -ErrorAction SilentlyContinue)) {
        Write-Host 'Audio backend: install mpv, or set WISP_MPV to the path of mpv.exe.'
    }
} finally {
    $resolvedScratch = [IO.Path]::GetFullPath($scratch)
    $resolvedTemp = [IO.Path]::GetFullPath([IO.Path]::GetTempPath()).TrimEnd('\') + '\'
    if ($resolvedScratch.StartsWith($resolvedTemp,[StringComparison]::OrdinalIgnoreCase)) { Remove-Item -LiteralPath $resolvedScratch -Recurse -Force }
}
