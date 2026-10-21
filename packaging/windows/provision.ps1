[CmdletBinding()]
param(
    [Parameter(Mandatory)][string]$ArchivePath,
    [Parameter(Mandatory)][string]$InstallDir,
    [Parameter(Mandatory)][string]$LogPath
)
$ErrorActionPreference = 'Stop'
# A caller using PowerShell 7 may pass its incompatible module directories to
# Windows PowerShell. Use this child's inbox modules for checksum/ZIP cmdlets.
$env:PSModulePath = Join-Path $PSHOME 'Modules'
$recording = $false
$result = 0
try {
    [void][IO.Directory]::CreateDirectory([IO.Path]::GetDirectoryName($LogPath))
    Start-Transcript -LiteralPath $LogPath -Force | Out-Null
    $recording = $true
    & (Join-Path $PSScriptRoot 'install.ps1') -ArchivePath $ArchivePath -InstallDir $InstallDir -NoPath
} catch {
    Write-Error -ErrorRecord $_ -ErrorAction Continue
    $result = 1
} finally {
    if ($recording) { Stop-Transcript | Out-Null }
}
exit $result
