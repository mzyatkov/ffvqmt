// Package metrics orchestrates ffmpeg runs across multiple (file, metric) pairs.
package metrics

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/mzyatkov/ffvqmt/internal/ffmpeg"
)

// RunRequest is sent from the UI to the backend to start a calculation.
type RunRequest struct {
	RefPath       string   `json:"refPath"`
	DistPaths     []string `json:"distPaths"`
	Metrics       []string `json:"metrics"` // PSNR/SSIM/VMAF/XPSNR
	Skip          float64  `json:"skip"`
	Duration      float64  `json:"duration"`
	Scaling       string   `json:"scaling"`
	VMAFModel     string   `json:"vmafModel"`
	VMAFPool      string   `json:"vmafPool"`
	VMAFSubsample int      `json:"vmafSubsample"`
	VMAFPhone     bool     `json:"vmafPhone"`
	VMAFUpscale   bool     `json:"vmafUpscale"`
	LogCommands   bool     `json:"logCommands"`
	LogFrames     bool     `json:"logFrames"`
	LogFramesDir  string   `json:"logFramesDir"`
	TempDir       string   `json:"tempDir"`
	NThreads      int      `json:"nThreads"`

	AutoSaveResults     bool   `json:"autoSaveResults"`
	AutoSaveResultsFile string `json:"autoSaveResultsFile"`
}

// EmitFunc emits an event to the UI.
type EmitFunc func(event string, payload any)

// SummaryKey is the (file, metric) pair used to map summaries.
type SummaryKey struct {
	File   string
	Metric string
}

// Runner executes a RunRequest sequentially and emits progress events.
type Runner struct {
	prober *ffmpeg.Prober
	info   *ffmpeg.ProbeInfo
	emit   EmitFunc

	mu        sync.Mutex
	summaries map[SummaryKey]*ffmpeg.Summary
	cmdLog    *os.File
}

// NewRunner builds a runner.
func NewRunner(p *ffmpeg.Prober, info *ffmpeg.ProbeInfo, emit EmitFunc) *Runner {
	return &Runner{
		prober:    p,
		info:      info,
		emit:      emit,
		summaries: map[SummaryKey]*ffmpeg.Summary{},
	}
}

// Summaries returns the collected per-(file,metric) summaries (call after Run).
func (r *Runner) Summaries() map[SummaryKey]*ffmpeg.Summary { return r.summaries }

// Run executes the whole request. Errors are aggregated and emitted, then returned.
func (r *Runner) Run(ctx context.Context, req RunRequest) error {
	ffPath, err := r.prober.FFmpegPath()
	if err != nil {
		return err
	}
	if req.LogCommands {
		f, err := os.OpenFile("FFvqmt.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err == nil {
			r.cmdLog = f
			defer f.Close()
			fmt.Fprintf(f, "\n===== %s =====\n", time.Now().Format(time.RFC3339))
		}
	}

	refInfo, err := r.prober.MediaInfo(req.RefPath)
	if err != nil {
		return fmt.Errorf("probe reference: %w", err)
	}
	r.emit("mediainfo", map[string]any{"file": req.RefPath, "info": refInfo, "kind": "reference"})

	modelW, modelH := vmafModelResolution(req.VMAFModel)

	total := len(req.DistPaths) * len(req.Metrics)
	done := 0
	var firstErr error

	for _, dist := range req.DistPaths {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		distInfo, err := r.prober.MediaInfo(dist)
		if err != nil {
			r.emit("file:error", map[string]any{"file": dist, "error": err.Error()})
			continue
		}
		r.emit("mediainfo", map[string]any{"file": dist, "info": distInfo, "kind": "distorted"})

		for _, m := range req.Metrics {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			kind := ffmpeg.MetricKind(strings.ToUpper(m))
			if !r.supports(kind) {
				r.emit("metric:skipped", map[string]any{"file": dist, "metric": kind, "reason": "ffmpeg has no " + string(kind)})
				done++
				r.emit("run:progress", map[string]any{"percent": pct(done, total)})
				continue
			}
			err := r.runOne(ctx, ffPath, req, dist, refInfo, distInfo, kind, modelW, modelH)
			if err != nil && firstErr == nil {
				firstErr = err
			}
			done++
			r.emit("run:progress", map[string]any{"percent": pct(done, total)})
		}
	}
	return firstErr
}

func pct(done, total int) int {
	if total <= 0 {
		return 100
	}
	return done * 100 / total
}

func (r *Runner) supports(kind ffmpeg.MetricKind) bool {
	switch kind {
	case ffmpeg.MetricPSNR:
		return r.info.HasPSNR
	case ffmpeg.MetricSSIM:
		return r.info.HasSSIM
	case ffmpeg.MetricVMAF:
		return r.info.HasVMAF
	case ffmpeg.MetricXPSNR:
		return r.info.HasXPSNR
	}
	return false
}

func vmafModelResolution(model string) (int, int) {
	// Default Netflix HD model is 1920x1080. 4K models target 3840x2160.
	m := strings.ToLower(model)
	if strings.Contains(m, "4k") || strings.Contains(m, "uhd") {
		return 3840, 2160
	}
	return 1920, 1080
}

func (r *Runner) runOne(
	ctx context.Context,
	ffPath string,
	req RunRequest,
	dist string,
	refInfo, distInfo *ffmpeg.MediaInfo,
	kind ffmpeg.MetricKind,
	modelW, modelH int,
) error {
	id := fmt.Sprintf("%d-%s", time.Now().UnixNano(), strings.ToLower(string(kind)))
	tmpDir := req.TempDir
	if tmpDir == "" {
		tmpDir = os.TempDir()
	}
	opts := ffmpeg.BuildOptions{
		RefPath:        req.RefPath,
		DistPath:       dist,
		Skip:           req.Skip,
		Duration:       req.Duration,
		Scaling:        strings.ToLower(req.Scaling),
		VMAFModel:      req.VMAFModel,
		VMAFPool:       req.VMAFPool,
		VMAFSubsample:  req.VMAFSubsample,
		VMAFPhone:      req.VMAFPhone,
		VMAFUpscale:    req.VMAFUpscale,
		InBuildVMAF:    r.info.InBuildVMAF,
		RefWidth:       refInfo.Width,
		RefHeight:      refInfo.Height,
		DistWidth:      distInfo.Width,
		DistHeight:     distInfo.Height,
		ModelWidthMax:  modelW,
		ModelHeightMax: modelH,
		NThreads:       req.NThreads,
		TempDir:        tmpDir,
		ID:             id,
	}
	cmd, err := ffmpeg.Build(kind, opts)
	if err != nil {
		return err
	}
	defer os.Remove(cmd.TmpStat)

	if r.cmdLog != nil {
		fmt.Fprintf(r.cmdLog, "%s %s\n", ffPath, strings.Join(cmd.Args, " "))
	}
	r.emit("ffmpeg:cmd", map[string]any{
		"file":   dist,
		"metric": kind,
		"argv":   append([]string{ffPath}, cmd.Args...),
	})

	ex := exec.CommandContext(ctx, ffPath, cmd.Args...)
	stderr, err := ex.StderrPipe()
	if err != nil {
		return err
	}
	stdout, err := ex.StdoutPipe()
	if err != nil {
		return err
	}
	if err := ex.Start(); err != nil {
		return err
	}

	// Drain stdout to avoid blocking even though we use stats files
	go io.Copy(io.Discard, stdout)

	// Parse ffmpeg progress (time= speed=) from stderr; emit "time"
	go r.streamFFmpegStderr(stderr, dist, kind)

	if waitErr := ex.Wait(); waitErr != nil {
		var exitErr *exec.ExitError
		if errors.As(waitErr, &exitErr) {
			return fmt.Errorf("ffmpeg failed for %s/%s: %v", filepath.Base(dist), kind, waitErr)
		}
		return waitErr
	}

	// Parse stats file
	f, err := os.Open(cmd.TmpStat)
	if err != nil {
		return fmt.Errorf("open stats file: %w", err)
	}
	defer f.Close()

	emitFrame := func(fs ffmpeg.FrameSample) {
		r.emit("metric:frame", map[string]any{
			"file":   dist,
			"metric": kind,
			"frame":  fs.Frame,
			"value":  fs.Value,
			"extra":  fs.Extra,
		})
	}
	var sum *ffmpeg.Summary
	switch kind {
	case ffmpeg.MetricPSNR:
		sum, err = ffmpeg.ParsePSNR(f, emitFrame)
	case ffmpeg.MetricSSIM:
		sum, err = ffmpeg.ParseSSIM(f, emitFrame)
	case ffmpeg.MetricXPSNR:
		sum, err = ffmpeg.ParseXPSNR(f, emitFrame)
	case ffmpeg.MetricVMAF:
		sum, err = ffmpeg.ParseVMAFCSV(f, emitFrame)
	}
	if err != nil {
		return err
	}
	r.mu.Lock()
	r.summaries[SummaryKey{File: dist, Metric: string(kind)}] = sum
	r.mu.Unlock()
	r.emit("metric:summary", map[string]any{
		"file":    dist,
		"metric":  kind,
		"summary": sum,
	})

	// optional per-file frame CSV
	if req.LogFrames {
		dir := req.LogFramesDir
		if dir == "" {
			dir = "."
		}
		_ = os.MkdirAll(dir, 0o755)
		outPath := filepath.Join(dir, filepath.Base(dist)+"."+strings.ToLower(string(kind))+".csv")
		if err := copyAsCSV(cmd.TmpStat, outPath, kind); err != nil {
			r.emit("log:error", err.Error())
		}
	}
	return nil
}

func (r *Runner) streamFFmpegStderr(rd io.ReadCloser, file string, metric ffmpeg.MetricKind) {
	defer rd.Close()
	scanner := bufio.NewScanner(rd)
	scanner.Buffer(make([]byte, 64*1024), 512*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "frame=") || strings.Contains(line, "time=") {
			r.emit("ffmpeg:progress", map[string]any{
				"file":   file,
				"metric": metric,
				"line":   line,
			})
		}
	}
}

func copyAsCSV(src, dst string, kind ffmpeg.MetricKind) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	w := bufio.NewWriter(out)
	defer w.Flush()
	_, err = io.Copy(w, in)
	return err
}
