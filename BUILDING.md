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
| ffmpeg | 4.3+ (5.1+ recommended) | runtime dependency, **not bundled** |

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

Output:

| Platform | Path |
| --- | --- |
| Windows | `build/bin/FFvqmt.exe` |
| Linux   | `build/bin/FFvqmt` |
| macOS   | `build/bin/FFvqmt.app` |

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

FFvqmt calls an external `ffmpeg` binary; it is **not** bundled to keep
artifacts small and to let users choose a build with VMAF / XPSNR support.

Put `ffmpeg` (and `ffprobe`) into:

- the system `PATH`, or
- the same directory as the FFvqmt executable, or
- pass `-ffmpeg-dir=/path/to/dir` on the command line.

Recommended distributions:

- **Windows / Linux** — https://www.gyan.dev/ffmpeg/builds/ or https://github.com/BtbN/FFmpeg-Builds
- **macOS** — `brew install ffmpeg` or https://evermeet.cx/ffmpeg/

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
