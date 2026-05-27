package ffmpeg

import (
	"path/filepath"
	"strconv"
	"strings"
)

// RawFormat describes the raw-input parameters required for container-less
// formats like .yuv that don't carry resolution / pix_fmt / fps metadata.
//
// When all required fields (Format, PixFmt, Width, Height, FrameRate) are
// set, the values are translated into the corresponding ffmpeg pre-input
// flags (`-f rawvideo -pix_fmt ... -s WxH -framerate ...`) and prepended
// before the matching `-i` argument.
type RawFormat struct {
	Format    string  `json:"format"`    // ffmpeg demuxer, e.g. "rawvideo"
	PixFmt    string  `json:"pixFmt"`    // e.g. "yuv420p"
	Width     int     `json:"width"`     // pixels
	Height    int     `json:"height"`    // pixels
	FrameRate float64 `json:"frameRate"` // fps
}

// IsRawVideoPath returns true if path's extension is a known raw container
// that requires the user to specify dimensions / pix_fmt / fps.
func IsRawVideoPath(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".yuv", ".raw", ".rgb", ".bgr", ".gray", ".y":
		return true
	}
	return false
}

// DefaultRawFormat returns reasonable defaults for a given path extension.
// Returns nil if the file is not recognised as raw.
func DefaultRawFormat(path string) *RawFormat {
	if !IsRawVideoPath(path) {
		return nil
	}
	return &RawFormat{
		Format:    "rawvideo",
		PixFmt:    "yuv420p",
		Width:     1920,
		Height:    1080,
		FrameRate: 30,
	}
}

// inputArgs returns the ffmpeg / ffprobe pre-input flags for a raw format.
// Returns nil when the format is unspecified.
//
// Important: we use the *rawvideo demuxer option* names (`-pixel_format`,
// `-video_size`, `-framerate`) rather than the encoder-style `-pix_fmt`,
// `-s`, `-r` aliases. The short forms are recognised by `ffmpeg` (they map
// to AVCodecContext fields) but `ffprobe` rejects them with
// "Failed to set value '...' for option 'pix_fmt': Option not found"
// because ffprobe has no output codec context. The long forms work for
// both binaries.
func (r *RawFormat) inputArgs() []string {
	if r == nil {
		return nil
	}
	format := r.Format
	if format == "" {
		format = "rawvideo"
	}
	args := []string{"-f", format}
	if r.PixFmt != "" {
		args = append(args, "-pixel_format", r.PixFmt)
	}
	if r.Width > 0 && r.Height > 0 {
		args = append(args, "-video_size", strconv.Itoa(r.Width)+"x"+strconv.Itoa(r.Height))
	}
	if r.FrameRate > 0 {
		args = append(args, "-framerate", strconv.FormatFloat(r.FrameRate, 'f', -1, 64))
	}
	return args
}

// complete reports whether r carries enough information to be used as a
// drop-in replacement for ffprobe.
func (r *RawFormat) complete() bool {
	return r != nil && r.Width > 0 && r.Height > 0 && r.FrameRate > 0 && r.PixFmt != ""
}

// bytesPerPixel returns the (possibly fractional) number of bytes per pixel
// for the most common pixel formats. Returns 0 for unknown formats.
func bytesPerPixel(pf string) float64 {
	switch strings.ToLower(pf) {
	case "yuv420p", "yuvj420p", "nv12", "nv21":
		return 1.5
	case "yuv422p", "yuvj422p", "yuyv422", "uyvy422":
		return 2
	case "yuv444p", "yuvj444p":
		return 3
	case "yuv420p10le", "yuv420p12le", "yuv420p16le", "p010le":
		return 3
	case "yuv422p10le", "yuv422p12le", "yuv422p16le":
		return 4
	case "yuv444p10le", "yuv444p12le", "yuv444p16le":
		return 6
	case "gray", "y8":
		return 1
	case "gray10le", "gray12le", "gray16le":
		return 2
	case "rgb24", "bgr24":
		return 3
	case "rgba", "bgra", "argb", "abgr", "0rgb", "0bgr":
		return 4
	}
	return 0
}
