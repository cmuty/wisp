# Installation

## Requirements

- Windows x64; PowerShell 5.1 or newer for installation.
- Internet access for installation and online sources.
- A terminal that supports ANSI colors. Windows Terminal is recommended for Unicode and true color.
- A window at least 44 columns by 20 rows; around 100 by 35 gives the interface more room.

## Install

```powershell
irm https://github.com/cmuty/wisp/releases/latest/download/install.ps1 | iex
```

This runs a downloaded PowerShell script. To read it first:

```powershell
Invoke-WebRequest https://github.com/cmuty/wisp/releases/latest/download/install.ps1 -OutFile install-wisp.ps1
Get-Content .\install-wisp.ps1
.\install-wisp.ps1
```

The release installer pins an archive URL and SHA-256. It verifies the archive,
installs into `%LOCALAPPDATA%\Programs\wisp`, creates `bin\wisp.cmd`, and adds
that bin folder to your user PATH. Its helper downloads are pinned and checked
as well. mpv, yt-dlp, and, if needed, Deno are downloaded from their upstream projects.
The wisp release ZIP does not embed those helper executables.

Run `wisp` from any directory in PowerShell or CMD. A terminal that was already
open may need to be reopened to see the new PATH.

## Manual and local archive installation

Download the ZIP and `SHA256SUMS` from the [same release](https://github.com/cmuty/wisp/releases/latest).
Compare `Get-FileHash .\wisp-0.4.8-windows-amd64.zip -Algorithm SHA256` with the
published value. Extract to a folder of your choice, then run:

```powershell
.\setup-audio.ps1
.\wisp.exe
```

For a local source build, the source installer accepts:

```powershell
.\scripts\install.ps1 -ArchivePath .\dist\wisp-0.4.8-windows-amd64.zip
```

Use `-WithoutMpv -WithoutExtractor` to skip helper setup if managing those tools
yourself. Use `-NoPath` to avoid updating PATH. These parameters are available
when running a saved installer file, rather than piping it to `iex`.

## Update

Run the installation command again. It installs the new build and points the
launcher at it. Your profile is stored separately and is retained. Close and
reopen wisp to run the new version. Check with `wisp --version`.

## Remove

Close wisp. Remove `%LOCALAPPDATA%\Programs\wisp` and remove its `bin` entry
from your **user** PATH in Windows Environment Variables. To remove your
library, settings, and lyrics cache too, delete `%APPDATA%\wisp` (or your
custom `WISP_HOME` directory). Uninstalling does not delete your music files.
