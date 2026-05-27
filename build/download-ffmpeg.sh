#!/usr/bin/env bash
# Downloads a static ffmpeg+ffprobe with libvmaf and places them next to the
# built FFvqmt binary so the release is self-contained.
#
# Usage: build/download-ffmpeg.sh <target-dir> [linux|darwin-arm64|darwin-amd64]
# Called from wails.json postbuild hooks; can also be run manually.

set -euo pipefail

if [ "${FFVQMT_SKIP_FFMPEG:-}" = "1" ]; then
  echo "FFVQMT_SKIP_FFMPEG=1 — skipping ffmpeg bundling."
  exit 0
fi

TARGET="${1:-build/bin}"
PLATFORM="${2:-}"

if [ -z "$PLATFORM" ]; then
  case "$(uname -s)-$(uname -m)" in
    Linux-x86_64)        PLATFORM=linux ;;
    Darwin-arm64)        PLATFORM=darwin-arm64 ;;
    Darwin-x86_64)       PLATFORM=darwin-amd64 ;;
    *) echo "unsupported host: $(uname -s)-$(uname -m)"; exit 1 ;;
  esac
fi

mkdir -p "$TARGET"

# Skip re-download if already present (mtime > 1 day old triggers refresh).
if [ -x "$TARGET/ffmpeg" ] && [ -x "$TARGET/ffprobe" ]; then
  age=$(( $(date +%s) - $(stat -f %m "$TARGET/ffmpeg" 2>/dev/null || stat -c %Y "$TARGET/ffmpeg") ))
  if [ "$age" -lt 86400 ]; then
    echo "ffmpeg already in $TARGET (cached); skipping."
    "$TARGET/ffmpeg" -hide_banner -version | head -1
    exit 0
  fi
fi

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

case "$PLATFORM" in
  linux)
    url="https://github.com/BtbN/FFmpeg-Builds/releases/download/latest/ffmpeg-master-latest-linux64-lgpl.tar.xz"
    echo "Downloading $url"
    curl -fL "$url" -o "$tmp/ff.tar.xz"
    tar -xJf "$tmp/ff.tar.xz" -C "$tmp"
    cp "$tmp"/ffmpeg-*/bin/ffmpeg  "$TARGET/ffmpeg"
    cp "$tmp"/ffmpeg-*/bin/ffprobe "$TARGET/ffprobe"
    ;;
  darwin-arm64)
    # ffmpeg 8.1 — first osxexperts build that ships the `xpsnr` filter
    # (added upstream in 7.1). Older 7.0 builds silently disable the
    # XPSNR checkbox in the UI because the filter is not present.
    curl -fL https://www.osxexperts.net/ffmpeg81arm.zip  -o "$tmp/ff.zip"
    curl -fL https://www.osxexperts.net/ffprobe81arm.zip -o "$tmp/fp.zip"
    unzip -o "$tmp/ff.zip" -d "$TARGET" >/dev/null
    unzip -o "$tmp/fp.zip" -d "$TARGET" >/dev/null
    xattr -dr com.apple.quarantine "$TARGET/ffmpeg"  2>/dev/null || true
    xattr -dr com.apple.quarantine "$TARGET/ffprobe" 2>/dev/null || true
    ;;
  darwin-amd64)
    # ffmpeg 8.0 (osxexperts currently ships 8.0 for Intel, 8.1 for ARM).
    # Both include the `xpsnr` filter required by the XPSNR metric.
    curl -fL https://www.osxexperts.net/ffmpeg80intel.zip  -o "$tmp/ff.zip"
    curl -fL https://www.osxexperts.net/ffprobe80intel.zip -o "$tmp/fp.zip"
    unzip -o "$tmp/ff.zip" -d "$TARGET" >/dev/null
    unzip -o "$tmp/fp.zip" -d "$TARGET" >/dev/null
    xattr -dr com.apple.quarantine "$TARGET/ffmpeg"  2>/dev/null || true
    xattr -dr com.apple.quarantine "$TARGET/ffprobe" 2>/dev/null || true
    ;;
  *) echo "unknown platform: $PLATFORM"; exit 1 ;;
esac

chmod +x "$TARGET/ffmpeg" "$TARGET/ffprobe"

# macOS .app bundles the binary inside Contents/MacOS — copy there too.
# Look for the .app in the target dir (cwd-independent).
APP="$TARGET/FFvqmt.app"
if [ ! -d "$APP" ] && [ -d "build/bin/FFvqmt.app" ]; then
  APP="build/bin/FFvqmt.app"
fi
if [ -d "$APP" ]; then
  cp "$TARGET/ffmpeg"  "$APP/Contents/MacOS/ffmpeg"
  cp "$TARGET/ffprobe" "$APP/Contents/MacOS/ffprobe"
  chmod +x "$APP/Contents/MacOS/ffmpeg" "$APP/Contents/MacOS/ffprobe"
  xattr -dr com.apple.quarantine "$APP/Contents/MacOS/ffmpeg"  2>/dev/null || true
  xattr -dr com.apple.quarantine "$APP/Contents/MacOS/ffprobe" 2>/dev/null || true
fi

echo "✓ ffmpeg bundled into $TARGET"
"$TARGET/ffmpeg" -hide_banner -version | head -1
"$TARGET/ffmpeg" -hide_banner -filters 2>/dev/null | grep -E 'libvmaf|xpsnr|psnr|ssim' || true
