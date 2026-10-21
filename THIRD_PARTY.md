# Third-party notices

## Go dependencies

Exact module versions and checksums are recorded in `go.mod` and `go.sum`.
The release build copies available top-level LICENSE, LICENCE, COPYING, and NOTICE
files from the resolved dependency modules into `licenses/` in the ZIP.
Dependencies include Bubble Tea, Bubbles, Lip Gloss, go-winio, and their transitive
modules. Their individual licenses remain applicable alongside wisp's MIT license.

## External audio tools

Default wisp ZIPs do not contain mpv, yt-dlp, or Deno executables. The installer
downloads pinned upstream artifacts and checks their SHA-256 values. Their own
licenses and included component notices apply:

| Tool | Upstream and license information |
| --- | --- |
| mpv | [mpv-player/mpv](https://github.com/mpv-player/mpv), [copyright and build options](https://github.com/mpv-player/mpv/blob/master/Copyright) |
| yt-dlp | [yt-dlp/yt-dlp](https://github.com/yt-dlp/yt-dlp), including the license of its packaged executable and dependencies |
| Deno | [denoland/deno](https://github.com/denoland/deno), including third-party components |

mpv is controlled as a separate process through JSON IPC. The selected build's
license depends on its enabled components, including FFmpeg. If creating a
custom bundled distribution with `-MpvDirectory`, preserve its notices and
satisfy the corresponding source and redistribution obligations of that exact
build. A project link alone does not replace those obligations.

## Installer toolchain

The optional Windows EXE setup wizard is built with
[Inno Setup](https://jrsoftware.org/isinfo.php). Its compiler is provisioned on
the temporary build runner and is not included in the portable application ZIP.
The installer engine is third-party software supplied by that toolchain.

## Services and content

SoundCloud, Spotify, and YouTube Music names identify their respective services.
wisp is not affiliated with or endorsed by them. Public web integrations are
unofficial. LRCLIB provides online lyrics lookup; local LRC files are also supported.
Service terms and rights in music, metadata, and lyrics apply separately from
the code license. See [sources](docs/SOURCES.md).

Showcase tracks and lyrics in preview mode are demonstration fixtures. The banner
and UI captures in `docs/assets` are project artwork and are covered by the
repository's MIT license.
