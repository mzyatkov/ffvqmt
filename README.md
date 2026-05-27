# FFvqmt — Fast Forward Video Quality Measurement Tool

> Open-source, cross-platform GUI for measuring and visualizing video Visual
> Quality Metrics (PSNR, SSIM, VMAF, XPSNR). Built on top of Go + Wails + Vue 3.

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
![Platforms](https://img.shields.io/badge/platforms-Windows%20%7C%20Linux%20%7C%20macOS-blue)

FFmpeg can be used for calculating different visual quality metrics (PSNR, SSIM, VMAF, XPSNR).
FFvqmt is an FFmpeg GUI whose purpose is to visualize quality metrics calculated by FFmpeg.
The program allows you to select files without dealing with command line, calculate & visualize PSNR, SSIM, VMAF & XPSNR quality metrics for all of them in one go.

Well, and build shiny interactive graphs of course:

<p align="center"><img src="screenshots/screenshot-1_0_0.png" width="900"/></p>



## Features
- PSNR, SSIM, VMAF, XPSNR visual quality metrics;
- Processing up to 24 files in one go (this can be increased by configuring graphs' styles) ;
- No limitations on frame size for PSNR/SSIM/XPSNR, Full HD/4K for VMAF;
- Brief media info for reference and distorted files;
- Thumbnail for reference file;
- “Bad” frames can be extracted and saved as PNG images for further analisys;
- Only parts of video files can be analyzed;
- Easy to use UI: drag & drop files from Explorer onto Reference field and Files list or use file choosers;
- Frames graphs can be zoomed in/out with mouse wheel (try it over graph or graph's axes), panned with right mouse button and saved as SVG or PNG;
- Frames graphs are optional and can be turned off in menu or with command line option;
- Frames metrics can be saved as tab-delimited **csv** files and then opened in Excel;
- FFMpeg commands issued by FFvqmt can be saved to log file (`FFvqmt.log`);
- Average metrics, frames statistics, frame size, bitrate, date/time and file name can be saved to tab-delimited **csv** file (appended) and then opened in Excel;
- VMAF model selected automatically based on reference media info, but can be changed using UI;
- Supported VMAF models type (**in-build**, **json** or **pkl**) detected automatically;
- Most program options could be supplied as command line parameters;
- Free. No registration, banners, tracking etc.



## Requirements
- Windows/Linux/Macos arm/Macos x86-64
- VMAF metric require special FFMpeg's build. It is supported since FFMpeg version 4.3 (stable). However, you'd better use FFMpeg 4 or newer.
  In addition, VMAF model files must be in sub-folder `vmaf-models`. The most common models are already included in archive. You can get other models from [Netflix VMAF project](https://github.com/Netflix/vmaf/).
  If you use the most recent FFMpeg builds, the vmaf models could be **in-build** into it so you do not need to use separate vmaf models.



## How to use
- Unpack into a folder;
- Put FFMpeg.exe into the program folder or make it available through system %PATH%;
- Run the program;
- Use UI to add reference file and at least one distorted file (you can drag & drop files from Explorer or use file choosers);
- Select metrics you'd like to calculate;
- Click “Start” button.


## How to run with command line options

```
FFvqmt [options] ref.mp4 file1.mp4 [file2.mp4] [file3.mp4] [...]
```

On Windows the binary is `FFvqmt.exe`, on Linux/macOS just `FFvqmt`.

### Accepted options
    -auto-save-results                     After calculation save results to log
    -auto-save-results-file=<filespec>     Example: C:\path\file.csv
    -duration=<seconds>                    Duration of video stream to be processed
                                           Default: 0 (whole stream used)
    -exit                                  Exit after run
    -ffmpeg-dir=<dirspec>                  ffmpeg.exe location directory
                                           Default: not specified,
                                           so ffmpeg.exe expected to be in program directory or in %PATH%
    -log-commands                          Log ffmpeg commands
    -log-frames                            Log frames' metrics in csv files
    -log-frames-dir=<dirspec>              Directory where frame's metrics will be stored
    -log-level=DEBUG|ERROR|INFO            Default: INFO
    -metric=PSNR|SSIM|VMAF|XPSNR           Default: all (-metric=PSNR -metric=SSIM -metric=VMAF)
    -plot-update-speed=HIGH|NORMAL|LOW|OFF Plots update frequency
                                           Default: NORMAL
    -plot-window-manual                    Do not open plot window automatically when starting calculation
    -project=<filespec>                    Read project options from specified file
                                           Example: C:\path\to\project.ffvqmtproj
    -run                                   Run calculation when program started
    -scaling-method=NEIGHBOR|GAUSS|BILINEAR|BICUBIC|LANCZOS|SINC|SPLINE
                                           Default: BICUBIC
    -skip=<seconds>                        Duration of video stream to be skipped
                                           Default: 0 (stream processed from the beginning)
    -temp-dir=<dirspec>                    Directory where temporary files will be stored
                                           Default: default user temporary directory
    -vmaf-model=<filename>                 Default: detected automatically based on reference media info
    -vmaf-phone-model
    -vmaf-pool=MEAN|HARMONIC_MEAN          Default: MEAN
    -vmaf-subsample=<value>                1 means VMAF metric for every frame to be calculated
                                           2 means VMAF metric for every 2nd frame to be calculated
                                           And so on
                                           Default: 1
    -vmaf-upscale-to-model                 Upscale ref and distorted to model's resolution.
                                           To turn it **off** falsy value need to be specified (0 or 'false')
                                           Default: **on** (true)

All options can be provided using single leading dash (-option) or double leading dash (--option).


#### Examples

```bash
# Windows
FFvqmt.exe \\server\path\to\ref.mp4 "c:\path\to\my file.mp4"
FFvqmt.exe -log-frames -metric=SSIM -metric=VMAF -run c:\to\ref.mp4 c:\to1\file1.avi c:\to2\file2.avs
FFvqmt.exe -project=c:\path\to\project.ffvqmtproj -vmaf-model=vmaf_v0.6.1neg -vmaf-pool=harmonic_mean -run

# Linux / macOS
./FFvqmt /data/ref.mp4 /data/dist1.mp4 /data/dist2.mp4
./FFvqmt -log-frames -metric=SSIM -metric=VMAF -run /data/ref.mp4 /data/dist.mp4
./FFvqmt -project=~/projects/test.ffvqmtproj -run
```


## Limitations
- You have to be very careful and supply video files in the same colour range or with correct colour range's meta. Otherwise ffmpeg.exe could make incorrect transformation and give you incorrect results ([more details](https://www.vegascreativesoftware.info/us/forum/magicyuv-2-20-released--117638/?page=3#ca772279)). I do have plans to improve the program to prevent/minimize this happen.


## Troubleshooting
- Close FFvqmt and delete `FFvqmt.log`;
- Run the program with option `-log-level=debug`;
- In program menu activate “Options | Write FFMpeg commands to log” (required for version below 1.3.0);
- Add reference file;
- Add one distorted file and make it active (checkbox on the left of filename must be ticked);
- Make sure at least one metric is active (checkbox on the left of metric name must be ticked);
- Click “Start” button;
- Take screenshot (Alt+PrnScr or Win+Shift+S and paste it into image editor and save as PNG);
- Close the program;
- Analyze `FFvqmt.log`. You can try to run the very last ffmpeg command directly;
- Upload archived `FFvqmt.log` with screenshot to dropbox (or similar) and share the link.


### Common issues
1. “Start” button disabled
    - No FFMpeg.exe found;
    - No reference file added, reference file does not exist (red file name) or program is unable to get file's media info;
    - No distorted file added, distorted file does not exist (red file name), program unable to get file's media info or no active distorted files (files with checkbox on the left of the file name ticked);
    - No metrics selected.
2. VMAF checkbox disabled
    - FFMpeg.exe does not support VMAF. [Download](https://ffmpeg.org/download.html) newer version, make sure it supports VMAF.
    - No `vmaf-models` folder with supported models in it. FFMpeg might support **pkl** model but there are only **json** models in the folder or visa versa.
    - Program path contains non-English characters. FFvqmt itself should not have issues with non-English characters in paths, but FFMpeg.exe could fail while trying to open VMAF model file during FFMpeg feature-detection startup process.
3. Error while calculating VMAF metric
    - Invalid VMAF model file. The first thing that you should check if you downloaded models on you own. Model file must be less than 30KB and should not contain HTML in it.
4. I'm trying to calculate VMAF metric comparing the file with itself and **not** getting score 100.
    - Based on [VMAF FAQ](https://github.com/Netflix/vmaf/blob/master/resource/doc/faq.md#q-when-i-compare-a-video-with-itself-as-reference-i-expect-to-get-a-perfect-score-of-vmaf-100-but-what-i-see-is-a-score-like-987-is-there-a-bug) this is by design.


## Building from source

FFvqmt is a Go + [Wails v2](https://wails.io/) + Vue 3 + TypeScript application.
See [`BUILDING.md`](BUILDING.md:1) for full instructions; the short version is:

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@v2.9.2
cd frontend && npm install && cd ..
wails dev          # hot-reload dev mode
wails build        # native build for current platform
```

## Releases

Every git tag matching `v*` triggers
[`.github/workflows/release.yml`](.github/workflows/release.yml:1) which builds
all four artifacts in parallel and attaches them to a GitHub Release:

| Platform | File |
| --- | --- |
| Windows x86-64 | `FFvqmt-windows-amd64.zip` |
| Linux x86-64 | `FFvqmt-linux-amd64.tar.gz` + `FFvqmt-linux-amd64.AppImage` |
| macOS Apple Silicon | `FFvqmt-macos-arm64.dmg` |
| macOS Intel | `FFvqmt-macos-amd64.dmg` |

Each artifact ships with `vmaf-models/` and a SHA-256 checksum file.

## Contributing

Bug reports, feature requests, and pull requests are welcome — see
[`CONTRIBUTING.md`](CONTRIBUTING.md:1) for the contribution guide.

## License

FFvqmt is released under the [MIT License](LICENSE).

FFmpeg itself is licensed separately (LGPL/GPL); FFvqmt only invokes the
`ffmpeg` binary as an external process and does not link against its
libraries, so it has no impact on FFvqmt's MIT licensing of its own code.
The bundled VMAF models in [`vmaf-models/`](vmaf-models:1) come from the
[Netflix VMAF project](https://github.com/Netflix/vmaf/) and are licensed
under BSD-2-Clause.
