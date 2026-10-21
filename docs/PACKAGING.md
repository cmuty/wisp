# Windows packages

## EXE setup

Download [wisp-setup-x64.exe](https://github.com/cmuty/wisp/releases/latest/download/wisp-setup-x64.exe).
Releases also include a versioned filename for immutable package-manager URLs.
The English/Russian Inno Setup wizard installs per user into
`%LOCALAPPDATA%\Programs\wisp`, registers user PATH, adds a Start menu shortcut,
and registers an uninstaller in Windows Installed Apps. No elevation is needed.
The setup program is unsigned; its SHA-256 is in the release's `SHA256SUMS`.

The EXE embeds the verified wisp ZIP, not the external audio executables.
Initial helper setup requires internet access and fetches pinned, verified
upstream artifacts. Errors are logged to `%LOCALAPPDATA%\wisp\installer.log`.
Uninstall removes the registered launcher, PATH entry, and files installed by
that setup version. Your `%APPDATA%\wisp` profile and music files are retained.
Older builds left by a previous PowerShell installation may remain on disk.

Silent install and removal are supported:

```powershell
.\wisp-0.4.12-setup-x64.exe /VERYSILENT /SUPPRESSMSGBOXES /NORESTART
```

Use Windows Installed Apps or the installation directory's `unins000.exe` to
remove an EXE installation. Choose one package manager for a given installation;
mixing managers can leave multiple launchers on PATH.

## Scoop

Scoop and Git must be installed first. If Scoop is available but Git is missing,
run `scoop install git` before adding buckets.

```powershell
scoop bucket add extras
scoop bucket add wisp https://github.com/cmuty/wisp
scoop install wisp/wisp
```

The bucket lives in `bucket/wisp.json` in this repository. Scoop installs the
portable ZIP and creates its own `wisp` shim. Audio dependencies are managed by
Scoop: `extras/mpv`, `main/yt-dlp`, and `main/deno`. The first command makes the
mpv bucket available. wisp's PowerShell installer is not invoked by Scoop.

Update with `scoop update wisp`; remove with `scoop uninstall wisp`. The profile
in `%APPDATA%\wisp` stays independent of Scoop. Release automation refreshes the
bucket URL and checksum only after the EXE and Scoop smoke tests pass.

## WinGet

Package identifier: **`cmuty.wisp`**. Release assets include
`winget-manifests.zip` with version, locale, and installer YAML files pointing to
the immutable release EXE and its SHA-256. The manifest declares a per-user Inno
installer, silent modes, the `wisp` command, and the stable uninstall product code.

**Catalog status: [submitted as PR #457542](https://github.com/microsoft/winget-pkgs/pull/457542);
not yet approved or indexed.** Microsoft has requested the maintainer's CLA
confirmation in [this bot comment](https://github.com/microsoft/winget-pkgs/pull/457542#issuecomment-6304761872).
An artifact in this repository does not make the package searchable through
the community catalog. After a submission is accepted and indexed, the command
will be:

```powershell
winget install --id cmuty.wisp --exact --source winget
```

Maintainers can extract the manifest ZIP and run `winget validate` against its
version directory before opening a pull request to
[microsoft/winget-pkgs](https://github.com/microsoft/winget-pkgs).

## Release maintenance

`scripts/build.ps1` builds and tests the application archive.
`scripts/build-installer.ps1` compiles `packaging/windows/wisp.iss` with Inno
Setup 6. `scripts/build-manifests.ps1` generates both package-manager formats
from the actual archive and EXE checksums. `release.json` includes the EXE name,
hash, and internet requirement.

The release workflow provisions a pinned compiler on a temporary runner,
tests silent EXE install/uninstall without Node on PATH, then tests Scoop
installation, dependencies, checksum, shim, and uninstall. A loopback HTTP server
serves the same archive during the Scoop smoke test; production manifests use
the public GitHub release URL. The workflow publishes all assets and updates
the Scoop bucket. WinGet community submissions remain a separate review step.
