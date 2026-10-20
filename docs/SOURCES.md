# Sources and imports

wisp has public web adapters and optional official API adapters. Enabling a
source selects where to search; it does not grant account access. Provider
responses and availability can change independently of wisp.

## SoundCloud

The public adapter reads public search and playlist data and resolves a
playable public stream just before playback. Playlist imports preserve the
returned ordering and metadata as a local snapshot. Temporary stream URLs are
not a permanent library format.

Optional official API access uses a user-supplied token. A track can be blocked,
preview-only, region-limited, or otherwise unavailable. `/open` provides its
original page. Public web behavior is unofficial and can change without notice.

## YouTube Music

The public adapter searches public song metadata and imports public playlists.
yt-dlp resolves audio at playback time; mpv plays the resulting stream. This is
an unofficial web integration, not an approved audio-streaming YouTube API.
Some videos require sign-in, are unavailable in a region, or cannot be resolved.
An official metadata API key does not turn those videos into licensed audio streams.

## Spotify

Spotify supplies playlist metadata, not a full-audio URL for mpv. Public playlist
links are read through publicly exposed embed metadata. That representation may
contain only part of a large playlist (up to 100 entries in the current adapter).
Use a CSV export when the public representation is incomplete.

At playback, wisp can search YouTube for a matching recording using track
metadata. The sound comes from the matched provider, not from Spotify. A cover,
remix, or similarly named track can be a wrong match. `/match` and `/bind` offer
explicit local/SoundCloud matching; `/open` opens the original Spotify page.
Public mode does not promise unrestricted Spotify catalog search. Optional
official API requests need credentials and remain subject to API access limits.

## Local imports

Add individual files or a folder with `/add`. M3U/M3U8 imports support local paths;
CSV is a metadata interchange option. File imports do not copy the audio into
wisp's data folder. A playlist is an ordered snapshot, rather than a live sync
with your account. Likes and recent history are local to wisp.

CSV accepts comma or semicolon separators, a UTF-8 BOM, and a required `Track Name`
or `Title` column. Optional columns include `Artist Name(s)`, `Spotify URI` or
`URL`, and `Duration (ms)`. Imports are bounded to 16 MiB and 10,000 tracks.
For example:

```csv
Title,Artist,URL,Duration (ms)
A little room for sound,wisp,,180000
```

CSV rows are stored as Spotify-style metadata and resolved through matching,
even when a URL is omitted. They are not local file paths.

## Platform terms

Technical availability and permission to use or redistribute content are
different questions. Use material and access methods you are allowed to use.
wisp does not unlock DRM, paywalls, or private content. Streaming integrations
are subject to service terms and changes in access policy:

- [SoundCloud API guide](https://developers.soundcloud.com/docs/api/guide) and [API terms](https://developers.soundcloud.com/docs/api/terms-of-use).
- [Spotify Web API](https://developer.spotify.com/documentation/web-api) and [Developer Policy](https://developer.spotify.com/policy).
- [YouTube Terms](https://www.youtube.com/static?template=terms) and [API Developer Policies](https://developers.google.com/youtube/terms/developer-policies).

Service names and track metadata identify their respective platforms and authors.
No platform endorsement is implied. The MIT license covers wisp's code, not
recordings, lyrics, provider access rights, or external tools.
