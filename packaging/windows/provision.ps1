[CmdletBinding()]
param(
    [Parameter(Mandatory)][string]$ArchivePath,
    [Parameter(Mandatory)][string]$InstallDir,
    [Parameter(Mandatory)][string]$LogPath
)
$ErrorActionPreference = 'Stop'
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
