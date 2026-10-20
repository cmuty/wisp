# Commands and controls

## The input line

Plain text searches the enabled sources and local library. Paste a supported
playlist URL to import it. Commands start with `/`; suggestions update while
typing. Use Up/Down to choose a hint and Tab to complete it. Enter runs the
command. Quote names and paths containing spaces.

```text
midnight ambient
/add "C:\Music\Night walks"
/import "https://soundcloud.com/artist/sets/playlist" "Night walks"
/playlists open "Night walks"
```

The URL above illustrates the expected shape; replace it with a real playlist.

## Navigation and playback

| Key | Action |
| --- | --- |
| Tab / Shift+Tab | Next/previous section, unless completing a command |
| Up/Down, wheel | Browse one item at a time |
| Page Up / Page Down | Browse a page |
| Enter | Play selected track; toggle pause if it is already current |
| Shift+Enter | Toggle selected track's like |
| Left/Right | Seek back/forward 5 seconds |
| Shift+Left/Right | Seek back/forward 1 second |
| `-` / `+` | Lower/raise volume by 5 |
| Shift with `-` / `+` | Adjust volume by 1 |
| F2 | Toggle lyrics |
| Ctrl+P / Ctrl+N | Input history |
| Esc | Hide hints or go back |

Seek and volume shortcuts require an empty input, preserving normal text editing.
Terminals differ in which modified keys they forward. Use `/like`, `/seek`,
and `/volume` if a shortcut is unavailable.

Starting a list result puts its following tracks into the playback queue in
list order. End of track advances automatically. `/next` skips to the next entry.
An imported playlist opens in Playlists; select a playlist and press Enter to
open its tracks, then choose a track to play.

## Reference

| Command | Purpose |
| --- | --- |
| `/help` | All commands and controls |
| `/search words` | Search enabled providers and local tracks |
| `/add "path"` | Add a local audio file or folder |
| `/import URL ["name"]` | Import a supported playlist |
| `/import "file.m3u"` | Import local M3U/M3U8 paths |
| `/import "file.csv"` | Import track metadata from CSV |
| `/play [n]` | Play a numbered result or resume the queue |
| `/pause` | Toggle pause |
| `/next` | Next queued track |
| `/stop` | Stop playback |
| `/seek +15` | Relative seek in seconds; a negative value seeks back |
| `/volume 70` | Set volume between 0 and 100 |
| `/queue` | Show the queue |
| `/queue add n` | Add result n |
| `/queue remove n` | Remove queue entry n |
| `/queue clear` | Clear the queue |
| `/like [n]` | Toggle a track's local like |
| `/likes` | Open liked tracks |
| `/recent` | Open recently played tracks |
| `/playlists` | Open saved playlists |
| `/playlists new "name"` | Create an empty playlist |
| `/playlists save "name"` | Save the displayed list |
| `/playlists open "name"` | Open a playlist |
| `/playlists add "name" n` | Add result n to a playlist |
| `/playlists delete "name"` | Delete a local playlist |
| `/lyrics` | Toggle lyrics |
| `/lyrics offset 0.5` | Adjust lyric timing for the current recording |
| `/settings` | Theme, search sources, language, and butterfly visibility |
| `/themes` | Theme picker |
| `/profile` | Profile, sources, and connection status |
| `/profile name "name"` | Rename your local profile |
| `/sources` | Choose enabled sources |
| `/language en` or `/language ru` | Switch interface language |
| `/connect [source]` | Optional API credential entry |
| `/match [n]` | Look for local/SoundCloud candidates for a metadata track |
| `/bind n` | Confirm a match candidate |
| `/open [n]` | Open the original service page |
| `/sleep 30m` or `/sleep off` | Set or cancel the sleep timer |
| `/quit` | Save and exit |

## Lyrics

F2 opens lyrics. Timed lines follow mpv's playback position, including seeks
and pauses. Wheel or Up/Down browses manually; Home resumes following. F5 retries
a failed lookup. Plain lyrics remain readable when no timed version exists.

For local audio, place a same-name `.lrc` next to the audio file. For remixes,
edits, and reuploads, an online match may have different timing; use a suitable
local LRC or a timing offset. wisp does not fabricate synchronized lines when
timestamps are unavailable.
