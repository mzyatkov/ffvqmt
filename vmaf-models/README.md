# VMAF Models

Drop Netflix VMAF model files (`*.json` for libvmaf v2 or `*.pkl` for v1) into this
folder. The most common ones are:

| File | Use |
| --- | --- |
| `vmaf_v0.6.1.json` | HD content (default) |
| `vmaf_4k_v0.6.1.json` | UHD/4K content |
| `vmaf_v0.6.1neg.json` | NEG model (less sensitive to enhancement) |

Download from https://github.com/Netflix/vmaf/tree/master/model

FFvqmt auto-detects the format supported by your `ffmpeg` build and picks the
appropriate model based on the reference file resolution.

If you use a recent `ffmpeg` (>= 5.1), VMAF models are bundled inside the binary
and you do not need this folder — FFvqmt will detect that case too.
