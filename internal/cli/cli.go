// Package cli parses FFvqmt command-line options per the README spec.
package cli

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// LogLevel represents -log-level= values.
type LogLevel string

const (
	LogDebug LogLevel = "DEBUG"
	LogInfo  LogLevel = "INFO"
	LogError LogLevel = "ERROR"
)

// PlotSpeed represents -plot-update-speed= values.
type PlotSpeed string

const (
	PlotHigh   PlotSpeed = "HIGH"
	PlotNormal PlotSpeed = "NORMAL"
	PlotLow    PlotSpeed = "LOW"
	PlotOff    PlotSpeed = "OFF"
)

// Scaling represents -scaling-method= values.
type Scaling string

const (
	ScaleNeighbor Scaling = "NEIGHBOR"
	ScaleGauss    Scaling = "GAUSS"
	ScaleBilinear Scaling = "BILINEAR"
	ScaleBicubic  Scaling = "BICUBIC"
	ScaleLanczos  Scaling = "LANCZOS"
	ScaleSinc     Scaling = "SINC"
	ScaleSpline   Scaling = "SPLINE"
)

// VMAFPool represents -vmaf-pool= values.
type VMAFPool string

const (
	PoolMean     VMAFPool = "MEAN"
	PoolHarmonic VMAFPool = "HARMONIC_MEAN"
)

// Options holds parsed command-line options.
type Options struct {
	AutoSaveResults     bool      `json:"autoSaveResults"`
	AutoSaveResultsFile string    `json:"autoSaveResultsFile"`
	Duration            float64   `json:"duration"`
	Exit                bool      `json:"exit"`
	FFmpegDir           string    `json:"ffmpegDir"`
	LogCommands         bool      `json:"logCommands"`
	LogFrames           bool      `json:"logFrames"`
	LogFramesDir        string    `json:"logFramesDir"`
	LogLevel            LogLevel  `json:"logLevel"`
	Metrics             []string  `json:"metrics"`
	PlotUpdateSpeed     PlotSpeed `json:"plotUpdateSpeed"`
	PlotWindowManual    bool      `json:"plotWindowManual"`
	Project             string    `json:"project"`
	Run                 bool      `json:"run"`
	ScalingMethod       Scaling   `json:"scalingMethod"`
	Skip                float64   `json:"skip"`
	TempDir             string    `json:"tempDir"`
	VMAFModel           string    `json:"vmafModel"`
	VMAFPhoneModel      bool      `json:"vmafPhoneModel"`
	VMAFPool            VMAFPool  `json:"vmafPool"`
	VMAFSubsample       int       `json:"vmafSubsample"`
	VMAFUpscaleToModel  bool      `json:"vmafUpscaleToModel"`

	RefFile   string   `json:"refFile"`
	DistFiles []string `json:"distFiles"`

	ShowHelp    bool `json:"-"`
	ShowVersion bool `json:"-"`
}

// Default returns options populated with README defaults.
func Default() *Options {
	return &Options{
		LogLevel:           LogInfo,
		Metrics:            nil, // becomes [PSNR SSIM VMAF] if empty at run-time
		PlotUpdateSpeed:    PlotNormal,
		ScalingMethod:      ScaleBicubic,
		VMAFPool:           PoolMean,
		VMAFSubsample:      1,
		VMAFUpscaleToModel: true,
	}
}

// Parse parses args (without program name) into Options.
func Parse(args []string) (*Options, error) {
	o := Default()
	metricsSet := false
	for i := 0; i < len(args); i++ {
		raw := args[i]
		if !strings.HasPrefix(raw, "-") {
			// positional: ref then dist files
			if o.RefFile == "" {
				o.RefFile = raw
			} else {
				o.DistFiles = append(o.DistFiles, raw)
			}
			continue
		}
		key, value, hasEq := splitFlag(raw)
		switch key {
		case "h", "help":
			o.ShowHelp = true
		case "version":
			o.ShowVersion = true
		case "auto-save-results":
			o.AutoSaveResults = true
		case "auto-save-results-file":
			o.AutoSaveResultsFile = value
		case "duration":
			f, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid -duration: %v", err)
			}
			o.Duration = f
		case "exit":
			o.Exit = true
		case "ffmpeg-dir":
			o.FFmpegDir = value
		case "log-commands":
			o.LogCommands = true
		case "log-frames":
			o.LogFrames = true
		case "log-frames-dir":
			o.LogFramesDir = value
		case "log-level":
			o.LogLevel = LogLevel(strings.ToUpper(value))
		case "metric":
			if !metricsSet {
				o.Metrics = nil
				metricsSet = true
			}
			o.Metrics = append(o.Metrics, strings.ToUpper(value))
		case "plot-update-speed":
			o.PlotUpdateSpeed = PlotSpeed(strings.ToUpper(value))
		case "plot-window-manual":
			o.PlotWindowManual = true
		case "project":
			o.Project = value
		case "run":
			o.Run = true
		case "scaling-method":
			o.ScalingMethod = Scaling(strings.ToUpper(value))
		case "skip":
			f, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid -skip: %v", err)
			}
			o.Skip = f
		case "temp-dir":
			o.TempDir = value
		case "vmaf-model":
			o.VMAFModel = value
		case "vmaf-phone-model":
			o.VMAFPhoneModel = true
		case "vmaf-pool":
			o.VMAFPool = VMAFPool(strings.ToUpper(value))
		case "vmaf-subsample":
			n, err := strconv.Atoi(value)
			if err != nil {
				return nil, fmt.Errorf("invalid -vmaf-subsample: %v", err)
			}
			o.VMAFSubsample = n
		case "vmaf-upscale-to-model":
			if !hasEq || value == "" {
				o.VMAFUpscaleToModel = true
			} else {
				o.VMAFUpscaleToModel = parseBool(value)
			}
		default:
			return nil, errors.New("unknown option: -" + key)
		}
	}
	if len(o.Metrics) == 0 {
		o.Metrics = []string{"PSNR", "SSIM", "VMAF"}
	}
	return o, nil
}

func splitFlag(raw string) (key, value string, hasEq bool) {
	s := strings.TrimLeft(raw, "-")
	if i := strings.Index(s, "="); i >= 0 {
		return strings.ToLower(s[:i]), s[i+1:], true
	}
	return strings.ToLower(s), "", false
}

func parseBool(v string) bool {
	switch strings.ToLower(v) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	}
	return true
}

// Usage returns the human-readable help text.
func Usage() string {
	return `FFvqmt [options] ref file1 [file2] [...]

  -auto-save-results                     After calculation save results to log
  -auto-save-results-file=<filespec>     Example: C:\path\file.csv
  -duration=<seconds>                    Duration of video stream to be processed (default 0 = whole)
  -exit                                  Exit after run
  -ffmpeg-dir=<dirspec>                  ffmpeg.exe location directory
  -log-commands                          Log ffmpeg commands to FFvqmt.log
  -log-frames                            Log frames' metrics in csv files
  -log-frames-dir=<dirspec>              Directory where frame's metrics will be stored
  -log-level=DEBUG|ERROR|INFO            Default: INFO
  -metric=PSNR|SSIM|VMAF|XPSNR           Default: PSNR SSIM VMAF
  -plot-update-speed=HIGH|NORMAL|LOW|OFF Default: NORMAL
  -plot-window-manual                    Do not open plot window automatically
  -project=<filespec>                    Read project options from file
  -run                                   Run calculation when program started
  -scaling-method=NEIGHBOR|GAUSS|BILINEAR|BICUBIC|LANCZOS|SINC|SPLINE  Default: BICUBIC
  -skip=<seconds>                        Skip seconds from start
  -temp-dir=<dirspec>                    Temporary files directory
  -vmaf-model=<filename>                 Auto-detected by default
  -vmaf-phone-model                      Use phone model
  -vmaf-pool=MEAN|HARMONIC_MEAN          Default: MEAN
  -vmaf-subsample=<value>                Subsample step, default 1
  -vmaf-upscale-to-model[=true|false]    Default: true
  --help                                 Show this help
  --version                              Show version

All flags accept both -opt and --opt.`
}
