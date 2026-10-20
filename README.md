<div align="center">

**English** · [Русский](README.ru.md)

<img src="docs/assets/banner.svg" alt="wisp — a little room for sound" width="100%">

### Your music. A quieter interface.

A terminal music player for Windows, with a persistent command line,
flowing gradients, and a little butterfly that follows the sound.

[![Release](https://img.shields.io/github/v/release/cmuty/wisp?style=flat-square&color=87dcef)](https://github.com/cmuty/wisp/releases/latest)
[![Windows x64](https://img.shields.io/badge/Windows-x64-b5a3e8?style=flat-square)](#quick-start)
[![Checks](https://github.com/cmuty/wisp/actions/workflows/ci.yml/badge.svg)](https://github.com/cmuty/wisp/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/license-MIT-dca2e6?style=flat-square)](LICENSE)
[![Discord](https://img.shields.io/badge/Discord-join_the_community-76e8e5?style=flat-square)](https://discord.gg/wJaJHK2hFA)

[Quick start](#quick-start) · [How it works](#how-it-works) · [Showcase](#showcase) · [Documentation](docs/README.md) · [Discord](https://discord.gg/wJaJHK2hFA)

</div>

## Quick start

Run this in **PowerShell** on 64-bit Windows:

```powershell
irm https://github.com/cmuty/wisp/releases/latest/download/install.ps1 | iex
```

Then launch from any folder in PowerShell or Command Prompt:

```text
wisp
```

The installer downloads a verified release, installs the audio helpers, and adds
`wisp` to your user PATH. No administrator access is needed. Open a new terminal
if an existing Command Prompt has not picked up the PATH change.

On first launch, choose **English or Russian**, enter a name, pick a theme, and
enable your search sources. Your wisp profile stays on your computer.

Prefer a manual install? See [installation and updates](docs/INSTALLATION.md),
including how to inspect the script before running it.

## Showcase

<img src="docs/assets/player.png" alt="wisp library in the Aurora theme, with search results, navigation, and a gradient playback bar" width="100%">

<details>
<summary>Afterglow, lyrics, and first launch</summary>

<img src="docs/assets/afterglow.png" alt="wisp in the peach, rose, and lavender Afterglow theme" width="100%">

<img src="docs/assets/lyrics.png" alt="Synchronized lyrics with the current line highlighted" width="100%">

<img src="docs/assets/welcome.png" alt="First launch with the gradient W logo" width="100%">

</details>

These images are rendered from the application's own `--preview` mode with
sample tracks and invented lyrics. They show the actual TUI layout; terminal
fonts and color support can change its appearance.

## A little room for your music

- **One input.** Search in plain text, paste a playlist link, or type a slash command. Hints appear as you type; Tab completes them.
- **Your listening space.** Library, Playlists, Likes, and Recent share one interface. Start a track and the following results become its queue.
- **Keyboard and mouse.** Browse with the wheel, pause with Enter, seek with the arrows, and adjust volume with `+` and `-`.
- **Lyrics that keep time.** Follow the current line, seek without losing your place, and load a local `.lrc` when a service has no match.
- **Made for dark terminals.** Five themes, a gradient W, restrained accents, and an animated butterfly you can disable in `/settings`. ASCII and reduced-motion modes are available.
- **English and Russian.** Choose a language during setup or change it later with `/language`.

## How it works

**Find → queue → listen.** wisp searches the enabled providers and your local
library. Playlist imports save an ordered snapshot of metadata and links.
Playable audio is resolved when needed and sent to **mpv** through JSON IPC.
The app does not maintain a downloaded music collection.

| Source | What wisp supports |
| --- | --- |
| SoundCloud | Public search, playlist imports, and playback of available public streams. |
| YouTube Music | Public search and playlist imports; audio resolution through yt-dlp. |
| Spotify | Public playlist metadata import, then matching to a recording on another provider. No native Spotify audio streaming. |
| Local files | Add files or folders; import local M3U/M3U8 playlists and metadata CSV files. |

Public web adapters can work without a personal API token, but availability
depends on the provider, region, and recording. Spotify public imports can be
limited by the data exposed by its embed page. Private, restricted, or unavailable
content is not unlocked. See [sources and limitations](docs/SOURCES.md).

The interface is written in **Go** with **Bubble Tea**, **Bubbles**, and **Lip Gloss**.
PowerShell handles installation. mpv handles playback; yt-dlp resolves YouTube
audio; LRCLIB and local LRC files provide lyrics.

## Find your way around

| Input | Action |
| --- | --- |
| Text or `/search words` | Search enabled sources and local tracks |
| Tab / Shift+Tab | Switch sections; complete an active slash-command hint |
| ↑ / ↓ or mouse wheel | Browse tracks |
| Enter | Play selection; pause/resume the selected current track |
| Shift+Enter | Toggle a like; `/like` is the terminal-independent fallback |
| ← / → | Seek 5 seconds; Shift seeks 1 second |
| `-` / `+` | Volume by 5; Shift adjusts by 1 |
| F2 or `/lyrics` | Open or close lyrics |
| `/settings` | Change theme and search sources |
| `/help` | Show all commands |

Seeking and volume shortcuts apply with an empty input. Modified key support
depends on the terminal. The [command reference](docs/COMMANDS.md) includes
command alternatives and playlist examples.

## Documentation

| Guide | What you will find |
| --- | --- |
| [Installation](docs/INSTALLATION.md) | One-line setup, manual installation, updates, removal |
| [Commands](docs/COMMANDS.md) | Playback, playlists, likes, queue, lyrics, and shortcuts |
| [Sources](docs/SOURCES.md) | Provider behavior, Spotify matching, and limitations |
| [Connections](docs/CONNECTIONS.md) | Optional API credentials and local data |
| [Troubleshooting](docs/TROUBLESHOOTING.md) | Playback, missing lyrics, terminal keys, and PATH |
| [Development](docs/DEVELOPMENT.md) | Build, test, architecture, and release packaging |
| [Design](docs/DESIGN.md) | Palettes, typography, motion, and localization |

## Community

<div align="center">
<table><tr><td align="center">
<br>
<strong>A little room to build wisp together.</strong><br><br>
Share your setup, suggest a feature, or get help with the player.<br><br>
<a href="https://discord.gg/wJaJHK2hFA"><strong>Join the wisp Discord</strong></a><br><br>
</td></tr></table>
</div>

Bug reports and focused pull requests are welcome. Read [Contributing](CONTRIBUTING.md)
and [Security](SECURITY.md) before sharing sensitive information.

## License

wisp is licensed under [MIT](LICENSE). External audio helpers and Go dependencies
have their own licenses; see [third-party notices](THIRD_PARTY.md).
wisp is an independent project and is not affiliated with the music services it supports.
