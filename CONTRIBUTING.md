# Contributing to FFvqmt

Thanks for taking the time to contribute! FFvqmt is an open-source project
released under the [MIT License](LICENSE). Contributions of any size — bug
reports, ideas, docs, code — are welcome.

## Quick start

```bash
git clone https://github.com/mzyatkov/ffvqmt.git
cd ffvqmt

# Backend deps (Go 1.22+)
go install github.com/wailsapp/wails/v2/cmd/wails@v2.9.2
go mod download

# Frontend deps (Node 20+)
cd frontend && npm install && cd ..

# Run with hot reload
wails dev
```

See [`BUILDING.md`](BUILDING.md:1) for the full build/release pipeline.

## Reporting bugs

Open an issue with:

1. The version (`FFvqmt --version`) and OS.
2. Output of `ffmpeg -version`.
3. Steps to reproduce.
4. The relevant section of `FFvqmt.log` (run with `-log-level=debug -log-commands`).
5. A short video sample if the bug is content-dependent.

Please **redact** any private file paths or usernames before posting.

## Submitting code

1. Fork the repo and create a feature branch.
2. Run `go test ./...` and `go vet ./...` — both must be clean.
3. Run `npm run build` inside `frontend/` — TypeScript must compile.
4. Open a pull request with a clear description of what changes and why.
5. Keep PRs focused — one feature or fix per PR is much easier to review.

### Code style

| Language | Tool |
| --- | --- |
| Go | `gofmt`, `go vet`. Idiomatic standard-library style. |
| TypeScript | Vue 3 `<script setup lang="ts">`, strict TS. |
| Commits | Conventional Commits encouraged: `feat:`, `fix:`, `docs:`, `chore:`. |

### Areas that always need help

- Additional VMAF / XPSNR test corpora and parser fixtures.
- macOS / Windows code-signing automation in CI.
- Localisation of UI strings.
- Linux distribution packaging (`.deb`, `.rpm`, Flatpak).

## Code of Conduct

Be kind. Discrimination, harassment, and personal attacks have no place in
this project. Reports may be sent to the maintainers via a private email
listed in `wails.json`.
