$ErrorActionPreference = 'Stop'
if (!$env:RUNNER_TEMP) { throw 'Compiler provisioning is restricted to an isolated CI runner.' }
$directory = Join-Path $env:RUNNER_TEMP 'wisp-inno'
New-Item -ItemType Directory -Path $directory -Force | Out-Null
$installer = Join-Path $directory 'innosetup-6.7.3.exe'
Invoke-WebRequest https://github.com/jrsoftware/issrc/releases/download/is-6_7_3/innosetup-6.7.3.exe -OutFile $installer
if ((Get-FileHash -LiteralPath $installer -Algorithm SHA256).Hash -ne '9c73c3bae7ed48d44112a0f48e66742c00090bdb5bef71d9d3c056c66e97b732') { throw 'Inno Setup checksum mismatch.' }
$destination = Join-Path $directory 'compiler'
$process = Start-Process -FilePath $installer -ArgumentList @('/VERYSILENT','/SUPPRESSMSGBOXES','/NORESTART','/CURRENTUSER','/NOICONS',('/DIR="'+$destination+'"')) -WindowStyle Hidden -Wait -PassThru
if ($process.ExitCode -ne 0) { throw "Compiler setup failed: $($process.ExitCode)" }
$compiler = Join-Path $destination 'ISCC.exe'
if (!(Test-Path -LiteralPath $compiler)) { throw 'Compiler setup did not produce ISCC.exe.' }
"INNO_COMPILER=$compiler" | Out-File -FilePath $env:GITHUB_ENV -Encoding utf8 -Append
