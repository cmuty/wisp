# Connections and local data

The default public adapters do not require a personal API token. Enable providers
in `/settings` or `/sources`. See [sources](SOURCES.md) for what each adapter can do.

## Optional API credentials

`/connect soundcloud`, `/connect spotify`, and `/connect youtube` open hidden
credential entry. Credentials are held in process memory, not saved to the
library JSON or command history. They expire with their provider and must be
supplied again after restarting. There is no bundled client secret or automatic
OAuth refresh flow.

Advanced users can supply credentials through the environment. Treat them as
secrets and do not paste values into screenshots, issues, or shared shell history.

| Variable | Purpose |
| --- | --- |
| `WISP_SOUNDCLOUD_TOKEN` | Optional SoundCloud access token |
| `WISP_SPOTIFY_TOKEN` | Optional Spotify access token |
| `WISP_YOUTUBE_KEY` | Optional YouTube metadata API key |
| `WISP_OFFICIAL_API_ONLY=1` | Disable public web adapters |

## Data on your machine

The default data directory is `%APPDATA%\wisp`. `library.json` stores your local
profile, language, theme, enabled sources, tracks, queue, playlists, likes, and
recent history. The lyrics cache lives under `lyrics-cache`. Set `WISP_HOME` to
choose another data directory. No wisp cloud account or cloud library sync is used.

Searches and media requests still go to the selected providers. Online lyrics
lookups send identifying track metadata to LRCLIB. Opening a service link uses
your browser. Those services have their own privacy policies.

## Runtime configuration

| Variable | Purpose |
| --- | --- |
| `WISP_HOME` | Custom local data directory |
| `WISP_MPV` | Explicit path to mpv.exe |
| `WISP_YTDLP` | Explicit path to yt-dlp.exe |
| `WISP_ASCII=1` | ASCII fallback for terminal art |
| `WISP_REDUCED_MOTION=1` | Reduce decorative motion |

Set variables before launching wisp. Runtime binaries can also be discovered
from their installation paths or PATH. See [troubleshooting](TROUBLESHOOTING.md)
for missing helpers and terminal compatibility.
