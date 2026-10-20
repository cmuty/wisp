# Troubleshooting

## `wisp` is not recognized

Open a new terminal after installation. Run `where.exe wisp` in CMD or
`Get-Command wisp` in PowerShell. The normal launcher is
`%LOCALAPPDATA%\Programs\wisp\bin\wisp.cmd`; its directory must be in user PATH.
An older PowerShell alias can override PATH: inspect `Get-Command wisp -All`.
Use `wisp --version` to check which build starts.

## No sound or missing audio backend

Check volume in wisp, Windows Volume Mixer, and the selected output device.
The normal installer installs mpv and yt-dlp. For a manual ZIP, run its
`setup-audio.ps1`, or supply `WISP_MPV` and `WISP_YTDLP` with full executable paths.
Check that the tools start independently. `WISP_AUDIO_OUTPUT=null` is a test
setting and intentionally produces no sound; remove it for normal playback.

## A provider has no results or a track fails

Check `/settings` for enabled sources. Try another provider or `/open` to inspect
the original page. A web service can be temporarily down, rate-limited, blocked,
or changed. Some tracks are private, region-restricted, preview-only, or removed.
Spotify public mode imports playlist metadata; it is not unrestricted native
Spotify search or streaming. Do not share tokens when filing a report.

## Lyrics are missing, out of time, or return HTTP 503

A 503 means the remote lyrics service is unavailable. wisp retries transient
failures with backoff and uses a local/cache result when available, but cannot
repair the remote service. Press F5 to retry later. Not every recording has
lyrics or timestamps. Remixes and edits can match the wrong timing; use
`/lyrics offset 0.5`, or a same-name local `.lrc` next to a local audio file.
Home restores auto-follow after manual scrolling.

## Shift+Enter, colors, or symbols look wrong

Modified keys depend on terminal protocol support. Use `/like`, `/seek +5`,
`/seek -1`, and `/volume 70` as reliable alternatives. Try Windows Terminal and
a Unicode monospace font. Run `wisp --ascii` for simpler art. Widen the window
if navigation or labels are clipped. Use `WISP_REDUCED_MOTION=1` to quiet the butterfly.

## Reporting a bug

Include `wisp --version`, terminal name, Windows version, source type, and exact
steps. A public track URL can help if it is appropriate to share. Remove your
username, local paths, tokens, and cookies from logs and screenshots.
