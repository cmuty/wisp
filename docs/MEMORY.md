# Memory footprint

The README's memory figure comes from a local observation of **wisp 0.4.10**
on Windows x64 (OS version `10.0.26200.0`), recorded on **20 October 2026 UTC**.
The existing session was playing audio when collection began: read-only mpv
JSON IPC queries returned `pause=false` and `idle-active=false` immediately
before sampling. No playback controls were changed for the measurement.

The collection lasted **31 seconds** and captured **25 samples**. The script
waits one second between reads; process-tree enumeration adds overhead, so this
is not an exact 1 Hz trace. Only `mpv.exe` was observed among the audio helpers
in this window.

| Process counter | Mean | Observed peak |
| --- | ---: | ---: |
| wisp working set | 31.9 MiB | 32.5 MiB |
| Audio helper working sets (mpv) | 49.2 MiB | 49.2 MiB |
| Combined process working sets | 81.0 MiB | 81.6 MiB |
| wisp private committed bytes | 61.8 MiB | 62.4 MiB |

One MiB is 1,048,576 bytes. Values are rounded independently.

## What the numbers mean

Working set is resident process memory and includes shared pages. The combined
figure sums the sampled working sets; it does not deduplicate shared pages
between processes. Private committed bytes is a different counter, not the
amount of physical RAM currently resident. Task Manager views using different
counters can therefore show different numbers.

The audio engine runs as a separate process. A number for `wisp.exe` alone
does not describe the complete playback process group. The measurement includes
observed descendant mpv, yt-dlp, Deno, and Node processes when present. It excludes
the terminal host, the measurement script, unrelated processes, and helpers
that start and exit between samples. In particular, this observation does not
measure a separate search/import/extraction workload.

This is a session snapshot, not a memory ceiling or a comparison benchmark.
Track formats, buffers, lyrics, library size, source operations, and session
length can change memory use. The observation does not establish a CPU,
power-consumption, or long-term memory-growth claim.

## Data and figure

- [Raw samples and metadata](assets/memory-sample.json).
- [Memory figure](assets/memory.png), generated from the recorded counters.
- [Collection script](../scripts/measure-memory.ps1).
- [Figure renderer](../scripts/render-memory.mjs).

The public report contains counters, executable names, and measurement metadata.
It omits process IDs, local paths, usernames, song names, and credentials.

## Reproduce

Start wisp, play a track, and run the collector in PowerShell from the repository root:

```powershell
.\scripts\measure-memory.ps1 -DurationSeconds 30 -OutputPath .preview-cache/memory-sample.json
```

If more than one wisp instance is running, specify its PID with `-WispProcessId`.
The script follows that instance's descendants and records both the interface
and observed audio helpers. If the selected wisp process exits or restarts,
the collection fails rather than reporting it as zero memory.

To render a new figure, install `@resvg/resvg-js` outside the repository, set
`WISP_RESVG` to its package directory, and run:

```powershell
node scripts/render-memory.mjs .preview-cache/memory-sample.json
```

The renderer writes the figure to `docs/assets/memory.svg` and `memory.png`.
When publishing a new observation, review the conditions and update the raw
report and README values together.
