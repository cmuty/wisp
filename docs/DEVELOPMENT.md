# Development

## Build and check

Install the Go version declared in `go.mod`. On Windows, from the repository root:

```powershell
go test ./...
go vet ./...
go build -o dist/wisp.exe ./cmd/wisp
.\dist\wisp.exe
```

Node is only needed for optional preview rendering scripts. It is not required
for the Go application. Playback needs an external mpv build; YouTube resolution
also needs yt-dlp and a supported JavaScript runtime.

## Project layout

```text
cmd/wisp/          CLI entry point, flags, terminal title
internal/app/      Bubble Tea model, views, input, navigation, themes, animation
internal/command/  Slash-command parsing
internal/i18n/     English and Russian interface strings
internal/library/ Local library, playlists, likes, recent history, imports
internal/lyrics/   LRC parsing, online lookup, cache, retry handling
internal/player/   mpv lifecycle, JSON IPC, playback state
internal/source/   Search, provider metadata, imports, stream resolution
internal/terminal/ Windows terminal integration
scripts/           Build, installation, audio setup, preview rendering
docs/              User guides and showcase assets
```

The TUI owns UI state and processes asynchronous results through messages.
Provider adapters return metadata; the playback layer owns mpv and its IPC
connection. Settings and collection snapshots are persisted independently of
temporary stream URLs and credentials.

## Optional integration tests

```powershell
$env:WISP_TEST_MPV = 'C:\Tools\mpv\mpv.exe'
go test ./internal/player ./internal/app
```

These tests use a null audio output. Public-provider smoke tests are opt-in with
`WISP_TEST_PUBLIC=1` and depend on live services; keep deterministic unit tests
independent of the network.

## Preview assets

```powershell
.\dist\wisp.exe --preview butterfly-search --preview-theme aurora
.\dist\wisp.exe --preview lyrics --preview-theme afterglow
```

Preview mode renders the same views with sample data and does not start audio or
save a profile. `scripts/showcase.mjs` generates the README assets using
`@resvg/resvg-js` installed outside the repository. Set `WISP_RESVG` to that
package's directory, then run `node scripts/showcase.mjs dist/wisp.exe`.

## Release

```powershell
.\scripts\build.ps1 -Version 0.4.5 -ReleaseBaseUrl https://github.com/cmuty/wisp/releases/download/v0.4.5
```

The script tests, vets, builds, gathers dependency license notices, and creates
the ZIP, pinned installer, `SHA256SUMS`, and `release.json` in `dist`.
It excludes local research and development previews. External audio executables
are fetched by the installer rather than embedded in the default archive.

Tagged pushes (`v*`) run the Windows release workflow. Review the changes before
tagging; CI publishes the resulting assets on GitHub Releases. Keep credentials,
profiles, downloaded media, and unrelated application artifacts out of Git.
