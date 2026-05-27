// Package ffmpeg wraps ffmpeg / ffprobe binaries: detection, media info, thumbnails,
// and metric command construction.
package ffmpeg

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// Prober detects ffmpeg/ffprobe and probes feature support.
type Prober struct {
	Dir         string // optional explicit directory containing ffmpeg/ffprobe
	ffmpegPath  string
	ffprobePath string
}

// NewProber creates a prober that searches the given directory first, then PATH.
func NewProber(dir string) *Prober { return &Prober{Dir: dir} }

// ProbeInfo describes the detected ffmpeg binary capabilities.
type ProbeInfo struct {
	FFmpegPath  string   `json:"ffmpegPath"`
	FFprobePath string   `json:"ffprobePath"`
	Version     string   `json:"version"`
	Filters     []string `json:"filters"`
	HasPSNR     bool     `json:"hasPSNR"`
	HasSSIM     bool     `json:"hasSSIM"`
	HasVMAF     bool     `json:"hasVMAF"`
	HasXPSNR    bool     `json:"hasXPSNR"`
	VMAFModels  []string `json:"vmafModels"`
	InBuildVMAF bool     `json:"inBuildVMAF"`
}

// MediaInfo summarises a media file via ffprobe.
type MediaInfo struct {
	Path       string  `json:"path"`
	Format     string  `json:"format"`
	DurationS  float64 `json:"duration"`
	Bitrate    int64   `json:"bitrate"`
	Width      int     `json:"width"`
	Height     int     `json:"height"`
	PixFmt     string  `json:"pixFmt"`
	ColorRange string  `json:"colorRange"`
	FrameRate  float64 `json:"frameRate"`
	FrameCount int64   `json:"frameCount"`
	VideoCodec string  `json:"videoCodec"`
	AudioCodec string  `json:"audioCodec"`
	SizeBytes  int64   `json:"sizeBytes"`
	// IsRaw is true when the file is a header-less raw container (.yuv etc.)
	// that requires the user to specify Format / PixFmt / Width / Height /
	// FrameRate before any metric can be calculated.
	IsRaw bool `json:"isRaw"`
}

// resolve locates ffmpeg + ffprobe; cached on first call.
func (p *Prober) resolve() error {
	if p.ffmpegPath != "" {
		return nil
	}
	exeFF := "ffmpeg"
	exePR := "ffprobe"
	if runtime.GOOS == "windows" {
		exeFF += ".exe"
		exePR += ".exe"
	}
	tryFind := func(name string) string {
		// 1. Explicit -ffmpeg-dir wins.
		if p.Dir != "" {
			cand := filepath.Join(p.Dir, name)
			if _, err := os.Stat(cand); err == nil {
				return cand
			}
		}
		// 2. Bundled binary next to the executable (incl. macOS .app/Contents/MacOS
		//    and .app/Contents/Resources/ffmpeg).
		if exe, err := os.Executable(); err == nil {
			dir := filepath.Dir(exe)
			candidates := []string{
				filepath.Join(dir, name),
				filepath.Join(dir, "ffmpeg", name),
			}
			if runtime.GOOS == "darwin" {
				// dir is .../FFvqmt.app/Contents/MacOS
				candidates = append(candidates,
					filepath.Join(dir, "..", "Resources", name),
					filepath.Join(dir, "..", "Resources", "ffmpeg", name),
				)
			}
			for _, cand := range candidates {
				if _, err := os.Stat(cand); err == nil {
					return cand
				}
			}
		}
		// 3. System PATH.
		if x, err := exec.LookPath(name); err == nil {
			return x
		}
		// 4. macOS GUI apps don't inherit shell PATH — try Homebrew.
		if runtime.GOOS == "darwin" {
			for _, dir := range []string{"/opt/homebrew/bin", "/usr/local/bin"} {
				cand := filepath.Join(dir, name)
				if _, err := os.Stat(cand); err == nil {
					return cand
				}
			}
		}
		return ""
	}
	p.ffmpegPath = tryFind(exeFF)
	p.ffprobePath = tryFind(exePR)
	if p.ffmpegPath == "" {
		return errors.New("ffmpeg binary not found (set -ffmpeg-dir or add to PATH)")
	}
	if p.ffprobePath == "" {
		// ffmpeg-only builds are OK for metrics but media info will be limited
		p.ffprobePath = ""
	}
	return nil
}

// FFmpegPath returns the resolved ffmpeg binary path.
func (p *Prober) FFmpegPath() (string, error) {
	if err := p.resolve(); err != nil {
		return "", err
	}
	return p.ffmpegPath, nil
}

// FFprobePath returns the resolved ffprobe path (may be empty).
func (p *Prober) FFprobePath() (string, error) {
	if err := p.resolve(); err != nil {
		return "", err
	}
	return p.ffprobePath, nil
}

// Probe runs ffmpeg -version and -filters to detect capabilities.
func (p *Prober) Probe() (*ProbeInfo, error) {
	if err := p.resolve(); err != nil {
		return nil, err
	}
	info := &ProbeInfo{FFmpegPath: p.ffmpegPath, FFprobePath: p.ffprobePath}

	verOut, err := run(p.ffmpegPath, "-hide_banner", "-version")
	if err != nil {
		return nil, fmt.Errorf("ffmpeg -version: %w", err)
	}
	info.Version = firstLine(verOut)

	filtersOut, err := run(p.ffmpegPath, "-hide_banner", "-filters")
	if err != nil {
		return nil, fmt.Errorf("ffmpeg -filters: %w", err)
	}
	for _, line := range strings.Split(filtersOut, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		name := fields[1]
		info.Filters = append(info.Filters, name)
		switch name {
		case "psnr":
			info.HasPSNR = true
		case "ssim":
			info.HasSSIM = true
		case "libvmaf":
			info.HasVMAF = true
		case "xpsnr":
			info.HasXPSNR = true
		}
	}
	if info.HasVMAF {
		help, _ := run(p.ffmpegPath, "-hide_banner", "-h", "filter=libvmaf")
		info.InBuildVMAF = strings.Contains(help, "model=") && strings.Contains(help, "version=")
	}
	info.VMAFModels = discoverVMAFModels()
	return info, nil
}

func discoverVMAFModels() []string {
	candidates := []string{"vmaf-models"}
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "vmaf-models"))
	}
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(cwd, "vmaf-models"))
	}
	seen := map[string]bool{}
	var out []string
	for _, dir := range candidates {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			n := e.Name()
			ext := strings.ToLower(filepath.Ext(n))
			if ext == ".json" || ext == ".pkl" {
				abs := filepath.Join(dir, n)
				if !seen[abs] {
					seen[abs] = true
					out = append(out, abs)
				}
			}
		}
	}
	return out
}

// MediaInfo runs ffprobe -show_streams -show_format.
// When the file is a raw container and a RawFormat is provided, ffprobe is
// invoked with the matching pre-input flags so it can still report duration
// and frame counts; otherwise the call short-circuits and returns a stub
// MediaInfo with IsRaw=true so the UI can prompt for the missing details.
func (p *Prober) MediaInfo(path string) (*MediaInfo, error) {
	return p.MediaInfoRaw(path, nil)
}

// MediaInfoRaw is the raw-aware variant of MediaInfo.
func (p *Prober) MediaInfoRaw(path string, raw *RawFormat) (*MediaInfo, error) {
	if err := p.resolve(); err != nil {
		return nil, err
	}
	isRaw := IsRawVideoPath(path)
	if isRaw && !raw.complete() {
		// Return a minimal record so the UI can collect the missing
		// parameters from the user.
		m := &MediaInfo{Path: path, Format: "rawvideo", IsRaw: true}
		if st, err := os.Stat(path); err == nil {
			m.SizeBytes = st.Size()
		}
		if raw != nil {
			m.Width = raw.Width
			m.Height = raw.Height
			m.PixFmt = raw.PixFmt
			m.FrameRate = raw.FrameRate
		}
		return m, nil
	}
	if p.ffprobePath == "" {
		return nil, errors.New("ffprobe not found")
	}
	// Raw-input flags MUST come before the file path, AND before global
	// output options for some ffprobe versions to pick them up. We also
	// use `-v error` (not `quiet`) so an actual failure surfaces in the
	// returned error string instead of "exit status 1".
	args := []string{
		"-v", "error",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
	}
	if isRaw {
		args = append(args, raw.inputArgs()...)
	}
	args = append(args, path)
	out, err := run(p.ffprobePath, args...)
	if err != nil {
		return nil, fmt.Errorf("ffprobe %v: %w", args, err)
	}
	var probed struct {
		Format struct {
			Duration string `json:"duration"`
			BitRate  string `json:"bit_rate"`
			Size     string `json:"size"`
			Name     string `json:"format_name"`
		} `json:"format"`
		Streams []struct {
			CodecType  string `json:"codec_type"`
			CodecName  string `json:"codec_name"`
			Width      int    `json:"width"`
			Height     int    `json:"height"`
			PixFmt     string `json:"pix_fmt"`
			ColorRange string `json:"color_range"`
			AvgFrameR  string `json:"avg_frame_rate"`
			RFrameRate string `json:"r_frame_rate"`
			NbFrames   string `json:"nb_frames"`
		} `json:"streams"`
	}
	if err := json.Unmarshal([]byte(out), &probed); err != nil {
		return nil, err
	}
	m := &MediaInfo{Path: path, Format: probed.Format.Name}
	m.DurationS, _ = strconv.ParseFloat(probed.Format.Duration, 64)
	m.Bitrate, _ = strconv.ParseInt(probed.Format.BitRate, 10, 64)
	m.SizeBytes, _ = strconv.ParseInt(probed.Format.Size, 10, 64)
	for _, s := range probed.Streams {
		switch s.CodecType {
		case "video":
			if m.Width == 0 {
				m.Width = s.Width
				m.Height = s.Height
				m.PixFmt = s.PixFmt
				m.ColorRange = s.ColorRange
				m.VideoCodec = s.CodecName
				m.FrameRate = parseFraction(s.AvgFrameR, s.RFrameRate)
				m.FrameCount, _ = strconv.ParseInt(s.NbFrames, 10, 64)
			}
		case "audio":
			if m.AudioCodec == "" {
				m.AudioCodec = s.CodecName
			}
		}
	}
	m.IsRaw = isRaw
	if isRaw {
		// ffprobe sometimes can't determine duration for piped rawvideo;
		// fall back to size / (bytes_per_frame * fps) when possible.
		if m.DurationS == 0 && raw != nil && raw.complete() {
			if bpp := bytesPerPixel(raw.PixFmt); bpp > 0 && m.SizeBytes == 0 {
				if st, err := os.Stat(path); err == nil {
					m.SizeBytes = st.Size()
				}
			}
			if bpp := bytesPerPixel(raw.PixFmt); bpp > 0 && m.SizeBytes > 0 {
				frameBytes := float64(raw.Width*raw.Height) * bpp
				if frameBytes > 0 {
					frames := float64(m.SizeBytes) / frameBytes
					m.FrameCount = int64(frames)
					m.DurationS = frames / raw.FrameRate
				}
			}
		}
	}
	return m, nil
}

// Thumbnail generates a small PNG snapshot of path at 5% into the video
// and returns it as a data URI (data:image/png;base64,...) so it can be
// rendered by the WebView, which does not allow loading file:// URLs from
// the wails:// / http:// origin.
func (p *Prober) Thumbnail(path string) (string, error) {
	return p.ThumbnailRaw(path, nil)
}

// ThumbnailRaw is the raw-aware variant of Thumbnail.
func (p *Prober) ThumbnailRaw(path string, raw *RawFormat) (string, error) {
	if err := p.resolve(); err != nil {
		return "", err
	}
	if IsRawVideoPath(path) && !raw.complete() {
		return "", errors.New("raw video: please specify format, pixel format, size and frame rate")
	}
	info, err := p.MediaInfoRaw(path, raw)
	ts := 1.0
	if err == nil && info.DurationS > 1 {
		ts = info.DurationS * 0.05
		if ts < 0.5 {
			ts = 0.5
		}
	}
	outDir := filepath.Join(os.TempDir(), "ffvqmt-thumbs")
	_ = os.MkdirAll(outDir, 0o755)
	stamp := fmt.Sprintf("%d", time.Now().UnixNano())
	outPath := filepath.Join(outDir, sanitize(filepath.Base(path))+"-"+stamp+".png")
	args := []string{
		"-hide_banner", "-loglevel", "error",
	}
	isRawInput := IsRawVideoPath(path)
	// Non-raw containers support fast input-seek (`-ss` before `-i`).
	// Raw video has no index, the synthetic duration can easily be wrong,
	// and short .yuv files often have < 1 second of footage — seeking past
	// EOF makes ffmpeg silently produce no output file. Just take frame 0.
	if !isRawInput && ts > 0 {
		args = append(args, "-ss", strconv.FormatFloat(ts, 'f', 3, 64))
	}
	if isRawInput {
		args = append(args, raw.inputArgs()...)
	}
	args = append(args, "-i", path)
	args = append(args,
		"-frames:v", "1",
		"-vf", "scale=320:-1",
		"-y", outPath,
	)
	if _, err = run(p.ffmpegPath, args...); err != nil {
		return "", fmt.Errorf("ffmpeg thumbnail %v: %w", args, err)
	}
	defer os.Remove(outPath)
	data, err := os.ReadFile(outPath)
	if err != nil {
		// ffmpeg returned success but wrote nothing — usually means -ss
		// past EOF or invalid pixel-format / resolution combo for raw.
		return "", fmt.Errorf("thumbnail not produced (check raw size/pix_fmt/fps): %w", err)
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(data), nil
}

func sanitize(s string) string {
	r := regexp.MustCompile(`[^A-Za-z0-9._-]+`)
	return r.ReplaceAllString(s, "_")
}

func parseFraction(a, b string) float64 {
	for _, s := range []string{a, b} {
		if s == "" || s == "0/0" {
			continue
		}
		parts := strings.SplitN(s, "/", 2)
		if len(parts) == 2 {
			num, _ := strconv.ParseFloat(parts[0], 64)
			den, _ := strconv.ParseFloat(parts[1], 64)
			if den > 0 {
				return num / den
			}
		}
	}
	return 0
}

func firstLine(s string) string {
	if i := strings.Index(s, "\n"); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return strings.TrimSpace(s)
}

func run(name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		// ffmpeg writes useful info to stderr; surface both
		if stderr.Len() > 0 && stdout.Len() == 0 {
			return stderr.String(), fmt.Errorf("%v: %s", err, strings.TrimSpace(stderr.String()))
		}
		return stdout.String() + stderr.String(), err
	}
	if stdout.Len() == 0 && stderr.Len() > 0 {
		return stderr.String(), nil
	}
	return stdout.String(), nil
}
