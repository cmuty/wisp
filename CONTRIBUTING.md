# Contributing to wisp

Start with the [development guide](docs/DEVELOPMENT.md). For ideas and questions,
join [Discord](https://discord.gg/wJaJHK2hFA) or open a GitHub discussion or issue.

Keep changes focused. Describe the user-visible problem, the resulting behavior,
and how you checked it. Screenshots or `--preview` frames help for TUI changes.
Use descriptive commit titles such as `fix(player): keep queue advancing after seek`.

Before opening a pull request, run `go test ./...` and `go vet ./...` on Windows.
Add regression coverage for behavioral changes, especially input handling,
provider parsing, playlist ordering, and playback lifecycle.

Preserve English and Russian interface strings together. Do not commit profiles,
tokens, cookies, downloaded media, third-party application dumps, or runtime binaries.
Keep provider failures explicit and respect access restrictions.

Contributions are licensed under the project's MIT license.
