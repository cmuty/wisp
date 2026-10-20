[CmdletBinding()]
param([Parameter(Mandatory)][string]$Version)
$ErrorActionPreference = 'Stop'
if (!$env:RUNNER_TEMP) { throw 'Run this installation smoke test on an isolated GitHub runner.' }
$root = Split-Path -Parent $PSScriptRoot
$destination = Join-Path $env:RUNNER_TEMP 'wisp-setup-smoke'
$exe = Join-Path $root "dist/wisp-$Version-setup-x64.exe"
$profileDir = Join-Path $env:RUNNER_TEMP 'wisp-profile-smoke'
New-Item -ItemType Directory -Path $profileDir -Force | Out-Null
$marker = Join-Path $profileDir 'preserve.txt'
[IO.File]::WriteAllText($marker, 'profile must survive install and uninstall')
$oldPath = $env:Path
$oldHome = $env:WISP_HOME
try {
    $env:WISP_HOME = $profileDir
    $env:Path = "$env:SystemRoot\system32;$env:SystemRoot"
    $installer = Start-Process -FilePath $exe -ArgumentList @('/VERYSILENT','/SUPPRESSMSGBOXES','/NORESTART','/SP-',('/DIR="'+$destination+'"')) -WindowStyle Hidden -PassThru -Wait
    if ($installer.ExitCode -ne 0) {
        $log = Join-Path $env:LOCALAPPDATA 'wisp/installer.log'
        if (Test-Path $log) { Get-Content $log }
        throw "EXE installation failed: $($installer.ExitCode)"
    }
    $actual = & (Join-Path $destination 'bin/wisp.cmd') --version
    if ($actual -ne "wisp $Version") { throw "Installed version mismatch: $actual" }
    $release = Get-ChildItem (Join-Path $destination 'releases') -Directory | Select-Object -First 1
    foreach ($helper in @('mpv/mpv.exe','tools/yt-dlp.exe','tools/deno.exe')) {
        if (!(Test-Path -LiteralPath (Join-Path $release.FullName $helper))) { throw "Missing audio helper: $helper" }
    }
    $bin = Join-Path $destination 'bin'
    if (([Environment]::GetEnvironmentVariable('Path','User') -split ';') -notcontains $bin) { throw 'EXE did not register user PATH.' }
    $uninstaller = Get-ChildItem $destination -Filter 'unins*.exe' | Select-Object -First 1
    $removed = Start-Process -FilePath $uninstaller.FullName -ArgumentList @('/VERYSILENT','/SUPPRESSMSGBOXES','/NORESTART') -WindowStyle Hidden -PassThru -Wait
    if ($removed.ExitCode -ne 0) { throw "Uninstall failed: $($removed.ExitCode)" }
    if (Test-Path -LiteralPath (Join-Path $destination 'bin/wisp.cmd')) { throw 'Uninstall left the launcher behind.' }
    if (([Environment]::GetEnvironmentVariable('Path','User') -split ';') -contains $bin) { throw 'Uninstall left PATH entry behind.' }
    if ((Get-Content -LiteralPath $marker -Raw) -ne 'profile must survive install and uninstall') { throw 'Profile marker changed.' }
    Write-Output 'EXE install, audio setup without Node, PATH registration, version and uninstall passed.'
} finally {
    $env:Path = $oldPath
    $env:WISP_HOME = $oldHome
}
