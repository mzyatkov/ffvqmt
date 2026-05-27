package ffmpeg

import (
	"fmt"
	"strconv"
	"strings"
)

// MetricKind is one of the supported metrics.
type MetricKind string

const (
	MetricPSNR  MetricKind = "PSNR"
	MetricSSIM  MetricKind = "SSIM"
	MetricVMAF  MetricKind = "VMAF"
	MetricXPSNR MetricKind = "XPSNR"
)

// Scaling is the ffmpeg scaling flag name (lowercased).
type Scaling string

// MetricCmd describes a single ffmpeg invocation for one (dist, metric) pair.
type MetricCmd struct {
	Kind    MetricKind
	Args    []string // ffmpeg argv (without binary path)
	Note    string   // human-readable description (e.g. "subsample=2 phone_model=1")
	TmpStat string   // optional path to ffmpeg stats output file (if not stdout)
}

// BuildOptions is the set of options needed to build a metric command.
type BuildOptions struct {
	RefPath        string
	DistPath       string
	Skip           float64 // seconds, -ss
	Duration       float64 // seconds, -t
	Scaling        string  // ffmpeg sws flag
	VMAFModel      string  // path or in-build name
	VMAFPool       string  // "mean" / "harmonic_mean"
	VMAFSubsample  int
	VMAFPhone      bool
	VMAFUpscale    bool
	InBuildVMAF    bool
	RefWidth       int
	RefHeight      int
	DistWidth      int
	DistHeight     int
	ModelWidthMax  int // typically 1920 (HD) / 3840 (UHD) — used for upscale-to-model
	ModelHeightMax int
	StatsToStdout  bool   // when true, stats_file=- to stream over stdout (not all filters support it)
	NThreads       int    // 0 = auto
	LogCommands    bool   // not used here, but kept for symmetry
	TempDir        string // where to place stats files
	ID             string // unique id for tmp file names
}

// Build returns ffmpeg argv for given metric.
func Build(kind MetricKind, o BuildOptions) (*MetricCmd, error) {
	args := []string{"-hide_banner", "-nostdin"}
	if o.Skip > 0 {
		args = append(args, "-ss", fmtFloat(o.Skip))
	}
	args = append(args, "-i", o.DistPath)
	if o.Skip > 0 {
		args = append(args, "-ss", fmtFloat(o.Skip))
	}
	args = append(args, "-i", o.RefPath)
	if o.Duration > 0 {
		args = append(args, "-t", fmtFloat(o.Duration))
	}

	sws := o.Scaling
	if sws == "" {
		sws = "bicubic"
	}
	// Build common scaling/format chain: scale distorted to match ref pix_fmt+size
	// Choose target W,H based on metric (VMAF may upscale to model)
	tw, th := o.RefWidth, o.RefHeight
	if kind == MetricVMAF && o.VMAFUpscale && o.ModelWidthMax > 0 && o.ModelHeightMax > 0 {
		if o.RefWidth < o.ModelWidthMax && o.RefHeight < o.ModelHeightMax {
			tw, th = o.ModelWidthMax, o.ModelHeightMax
		}
	}
	if tw == 0 {
		tw = -2
	}
	if th == 0 {
		th = -2
	}

	filterChain, statFile, err := buildFilter(kind, o, tw, th, sws)
	if err != nil {
		return nil, err
	}

	args = append(args, "-lavfi", filterChain, "-an", "-sn", "-dn", "-f", "null")
	if o.NThreads > 0 {
		args = append(args, "-threads", strconv.Itoa(o.NThreads))
	}
	args = append(args, nullSink())

	return &MetricCmd{
		Kind:    kind,
		Args:    args,
		TmpStat: statFile,
		Note:    fmt.Sprintf("target=%dx%d sws=%s", tw, th, sws),
	}, nil
}

func buildFilter(kind MetricKind, o BuildOptions, tw, th int, sws string) (string, string, error) {
	// [0:v] = distorted, [1:v] = reference
	scaleD := fmt.Sprintf("[0:v]scale=%d:%d:flags=%s,setpts=PTS-STARTPTS[d]", tw, th, sws)
	scaleR := fmt.Sprintf("[1:v]scale=%d:%d:flags=%s,setpts=PTS-STARTPTS[r]", tw, th, sws)

	statFile := ""
	switch kind {
	case MetricPSNR:
		statFile = tmpStats(o, "psnr")
		flt := fmt.Sprintf("%s;%s;[d][r]psnr=stats_file=%s", scaleD, scaleR, escapePath(statFile))
		return flt, statFile, nil
	case MetricSSIM:
		statFile = tmpStats(o, "ssim")
		flt := fmt.Sprintf("%s;%s;[d][r]ssim=stats_file=%s", scaleD, scaleR, escapePath(statFile))
		return flt, statFile, nil
	case MetricXPSNR:
		statFile = tmpStats(o, "xpsnr")
		flt := fmt.Sprintf("%s;%s;[d][r]xpsnr=stats_file=%s", scaleD, scaleR, escapePath(statFile))
		return flt, statFile, nil
	case MetricVMAF:
		statFile = tmpStats(o, "vmaf.csv")
		opts := []string{
			"log_path=" + escapePath(statFile),
			"log_fmt=csv",
		}
		if o.VMAFSubsample > 1 {
			opts = append(opts, "n_subsample="+strconv.Itoa(o.VMAFSubsample))
		}
		if o.VMAFPool != "" {
			opts = append(opts, "pool="+strings.ToLower(o.VMAFPool))
		}
		if o.VMAFModel != "" {
			if o.InBuildVMAF && !strings.ContainsAny(o.VMAFModel, "/\\") {
				// in-build model: pass as version=
				model := o.VMAFModel
				if o.VMAFPhone {
					model += ":enable_transform=true"
				}
				opts = append(opts, "model=version="+model)
			} else {
				model := "path=" + escapePath(o.VMAFModel)
				if o.VMAFPhone {
					model += "\\:enable_transform=true"
				}
				opts = append(opts, "model="+model)
			}
		} else if o.VMAFPhone {
			opts = append(opts, "model=version=vmaf_v0.6.1\\:enable_transform=true")
		} else if o.InBuildVMAF {
			// Use built-in HD model as default when no model specified
			opts = append(opts, "model=version=vmaf_v0.6.1")
		}
		if o.NThreads > 0 {
			opts = append(opts, "n_threads="+strconv.Itoa(o.NThreads))
		}
		flt := fmt.Sprintf("%s;%s;[d][r]libvmaf=%s", scaleD, scaleR, strings.Join(opts, ":"))
		return flt, statFile, nil
	}
	return "", "", fmt.Errorf("unsupported metric: %s", kind)
}

func tmpStats(o BuildOptions, suffix string) string {
	dir := o.TempDir
	if dir == "" {
		dir = "."
	}
	id := o.ID
	if id == "" {
		id = "stats"
	}
	return dir + string(pathSep()) + "ffvqmt-" + id + "-" + suffix + ".log"
}

func pathSep() rune { return separator }

// escapePath escapes characters that would otherwise be parsed by ffmpeg's lavfi syntax.
func escapePath(p string) string {
	// Escape ':' and '\' in path for lavfi
	r := strings.NewReplacer(`\`, `\\`, `:`, `\:`, `'`, `\'`)
	return r.Replace(p)
}

func nullSink() string {
	if separator == '\\' {
		return "NUL"
	}
	return "/dev/null"
}

func fmtFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', 3, 64)
}
