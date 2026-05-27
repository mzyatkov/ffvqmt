# Building FFvqmt

FFvqmt is a [Wails v2](https://wails.io/) desktop app:
**Go backend ↔ Vue 3 + TypeScript frontend**, packaged into a single native
executable on Windows, Linux and macOS.

## 1. One-time prerequisites

| Tool | Min version | Used for |
| --- | --- | --- |
| Go | 1.22 | backend + Wails build orchestration |
| Node.js | 20 | frontend (Vite + Vue) |
| Wails CLI | 2.9.2 | platform packaging |
| ffmpeg | 4.3+ (5.1+ recommended) | local dev only — release builds bundle a static ffmpeg+libvmaf |

Install Wails CLI once:

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@v2.9.2
```

Platform-specific:

- **Linux** — `sudo apt install libgtk-3-dev libwebkit2gtk-4.1-dev libsoup-3.0-dev pkg-config build-essential file`
- **macOS** — Xcode Command Line Tools (`xcode-select --install`)
- **Windows** — WebView2 (already on Win 11; for Win 10 see [WebView2 runtime](https://developer.microsoft.com/en-us/microsoft-edge/webview2/))

Verify everything is wired up:

```bash
wails doctor
```

## 2. Local development

```bash
# fetch Go deps
go mod download

# fetch frontend deps
cd frontend && npm install && cd ..

# hot-reload dev server (Vite + Wails)
wails dev
```

`wails dev` opens an Electron-like window with HMR. Backend changes recompile
Go automatically. Frontend changes hot-reload instantly.

## 3. Production build (current platform)

```bash
wails build -clean -trimpath -ldflags "-s -w"
```

The `postbuild:<platform>` hook in [`wails.json`](wails.json:1) runs
[`build/download-ffmpeg.sh`](build/download-ffmpeg.sh:1) (or
[`build/download-ffmpeg.ps1`](build/download-ffmpeg.ps1:1) on Windows) which
downloads a static `ffmpeg` + `ffprobe` with `libvmaf` and `xpsnr` and places
them next to the binary (and inside `FFvqmt.app/Contents/MacOS` on macOS).
The downloaded binaries are cached for 24h to avoid re-downloading on every
build; delete `build/bin/ffmpeg*` to force a refresh.

Output:

| Platform | Path | Bundled files |
| --- | --- | --- |
| Windows | `build/bin/FFvqmt.exe` | `ffmpeg.exe`, `ffprobe.exe` |
| Linux   | `build/bin/FFvqmt`     | `ffmpeg`, `ffprobe` |
| macOS   | `build/bin/FFvqmt.app` | `Contents/MacOS/ffmpeg`, `Contents/MacOS/ffprobe` |

Skip ffmpeg bundling for a faster dev iteration by running with the env var
`FFVQMT_SKIP_FFMPEG=1` — the app will fall back to the system `ffmpeg` on
PATH at runtime.

## 4. Cross-platform release builds

All four targets are built in CI by [`.github/workflows/release.yml`](.github/workflows/release.yml:1)
on every tag matching `v*`:

| Runner | Target | Artifact |
| --- | --- | --- |
| `windows-latest` | `windows/amd64` | `FFvqmt-windows-amd64.zip` |
| `ubuntu-22.04`   | `linux/amd64`   | `FFvqmt-linux-amd64.tar.gz` + `.AppImage` |
| `macos-14`       | `darwin/arm64`  | `FFvqmt-macos-arm64.dmg` |
| `macos-13`       | `darwin/amd64`  | `FFvqmt-macos-amd64.dmg` |

SHA-256 sums are uploaded alongside each artifact and attached to the GitHub
Release.

### Trigger a release

```bash
git tag v2.0.0
git push origin v2.0.0
```

### Manual / local cross-compile

Wails handles the heavy lifting:

```bash
wails build -platform windows/amd64
wails build -platform linux/amd64
wails build -platform darwin/arm64
wails build -platform darwin/amd64
```

Note: cross-compiling **out** of macOS for macOS works without CGo issues; from
Linux to Windows works with `mingw-w64`; from Linux to macOS is unsupported by
Wails — use the CI matrix instead.

## 5. About FFmpeg

Release artifacts **bundle** a statically-linked `ffmpeg` + `ffprobe` with
`libvmaf` (and XPSNR) already enabled, so the app works on a clean machine
without installing anything:

| Platform | Source |
| --- | --- |
| Windows / Linux | [BtbN/FFmpeg-Builds](https://github.com/BtbN/FFmpeg-Builds) — `*-lgpl` static build (libvmaf, xpsnr; without x264/x265) |
| macOS (arm64/amd64) | [osxexperts.net](https://www.osxexperts.net/) — static builds with libvmaf |

Lookup order at runtime (see [`prober.tryFind()`](internal/ffmpeg/prober.go:74)):

1. `-ffmpeg-dir=/path/to/dir` CLI flag
2. Directory of the FFvqmt executable (Windows/Linux) or
   `FFvqmt.app/Contents/MacOS` and `…/Contents/Resources` (macOS) — **bundle**
3. System `PATH`
4. Homebrew defaults on macOS (`/opt/homebrew/bin`, `/usr/local/bin`)

For local dev builds (`wails dev` / `wails build` without CI), install ffmpeg
yourself — `brew install ffmpeg`, `apt install ffmpeg`, or a BtbN release.

## 6. Code signing

- **Windows** — sign `FFvqmt.exe` with `signtool` using your own
  Authenticode certificate after CI download.
- **macOS** — set repo secrets `MACOS_CERT_P12_BASE64` and `MACOS_CERT_PASS`
  to enable signing/notarization (TODO in the workflow). Without them, users
  get an unsigned `.dmg`: right-click → Open the first time.
- **Linux** — AppImage signing via `gpg --detach-sign --armor` (optional).

## 7. Layout

```
ffvqmt/
├── main.go                  # Wails entry, CLI parsing
├── app.go                   # Wails-bound methods
├── internal/
│   ├── cli/                 # flag parser (matches README)
│   ├── ffmpeg/              # probe, command builder, parsers
│   ├── metrics/             # orchestration runner
│   ├── project/             # .ffvqmtproj read/write
│   ├── csvlog/              # aggregate CSV append
│   └── badframes/           # PNG extraction
├── frontend/                # Vue 3 + TS + Pinia + Plotly
├── vmaf-models/             # bundled VMAF JSON/PKL models
├── build/                   # Wails build assets (icon etc.)
└── .github/workflows/       # CI cross-platform release
```
