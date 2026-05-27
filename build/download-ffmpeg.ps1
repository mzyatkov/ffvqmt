# Downloads static ffmpeg + ffprobe (BtbN LGPL build with libvmaf) and places
# them next to FFvqmt.exe so the release is self-contained.
#
# Usage: pwsh build/download-ffmpeg.ps1 [TargetDir]

param(
    [string]$TargetDir = "build/bin"
)

$ErrorActionPreference = "Stop"

if ($env:FFVQMT_SKIP_FFMPEG -eq "1") {
    Write-Host "FFVQMT_SKIP_FFMPEG=1 - skipping ffmpeg bundling."
    exit 0
}

New-Item -ItemType Directory -Force -Path $TargetDir | Out-Null

$ffmpeg  = Join-Path $TargetDir "ffmpeg.exe"
$ffprobe = Join-Path $TargetDir "ffprobe.exe"

if ((Test-Path $ffmpeg) -and (Test-Path $ffprobe)) {
    $age = (Get-Date) - (Get-Item $ffmpeg).LastWriteTime
    if ($age.TotalDays -lt 1) {
        Write-Host "ffmpeg already in $TargetDir (cached); skipping."
        & $ffmpeg -hide_banner -version | Select-Object -First 1
        exit 0
    }
}

$url = "https://github.com/BtbN/FFmpeg-Builds/releases/download/latest/ffmpeg-master-latest-win64-lgpl.zip"
$tmp = New-TemporaryFile
$zip = "$($tmp.FullName).zip"
Remove-Item $tmp -Force

Write-Host "Downloading $url"
Invoke-WebRequest -Uri $url -OutFile $zip

$extract = Join-Path ([System.IO.Path]::GetTempPath()) "ffvqmt-ffmpeg-$([guid]::NewGuid())"
Expand-Archive -Path $zip -DestinationPath $extract -Force

Copy-Item (Join-Path $extract "ffmpeg-*\bin\ffmpeg.exe")  $ffmpeg  -Force
Copy-Item (Join-Path $extract "ffmpeg-*\bin\ffprobe.exe") $ffprobe -Force

Remove-Item $zip -Force
Remove-Item $extract -Recurse -Force

Write-Host "✓ ffmpeg bundled into $TargetDir"
& $ffmpeg -hide_banner -version | Select-Object -First 1
& $ffmpeg -hide_banner -filters 2>$null | Select-String -Pattern 'libvmaf|xpsnr'
