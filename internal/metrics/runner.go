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
	StartFrame    int      `json:"startFrame"` // optional; when >0 overrides Skip
	EndFrame      int      `json:"endFrame"`   // optional; when >StartFrame overrides Duration
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

	// Convert frame-based range to seconds, using the reference frame rate.
	// Frame range, when provided, overrides time-based Skip/Duration.
	if refInfo.FrameRate > 0 {
		if req.StartFrame > 0 {
			req.Skip = float64(req.StartFrame) / refInfo.FrameRate
		}
		if req.EndFrame > 0 && req.EndFrame > req.StartFrame {
			req.Duration = float64(req.EndFrame-req.StartFrame) / refInfo.FrameRate
		}
	}

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

	// DEBUG: Log VMAF-specific diagnostics
	if kind == ffmpeg.MetricVMAF {
		r.emit("debug:vmaf", map[string]any{
			"vmafModel":     req.VMAFModel,
			"inBuildVMAF":   r.info.InBuildVMAF,
			"refSize":       fmt.Sprintf("%dx%d", refInfo.Width, refInfo.Height),
			"distSize":      fmt.Sprintf("%dx%d", distInfo.Width, distInfo.Height),
			"modelTarget":   fmt.Sprintf("%dx%d", modelW, modelH),
			"vmafUpscale":   req.VMAFUpscale,
			"nThreads":      req.NThreads,
			"vmafPhone":     req.VMAFPhone,
			"vmafPool":      req.VMAFPool,
			"vmafSubsample": req.VMAFSubsample,
		})
	}
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

	emitFrame := func(fs ffmpeg.FrameSample) {
		r.emit("metric:frame", map[string]any{
			"file":   dist,
			"metric": kind,
			"frame":  fs.Frame,
			"value":  fs.Value,
			"extra":  fs.Extra,
		})
	}

	// Tail the stats file in real time so per-frame events stream to the UI
	// while ffmpeg is still running. PSNR/SSIM/XPSNR write line-by-line.
	// libvmaf flushes only at the end — in that case the tail will simply
	// deliver everything in one final burst, which is still correct.
	var ffmpegErr error
	ffmpegDone := make(chan struct{})
	go func() {
		ffmpegErr = ex.Wait()
		close(ffmpegDone)
	}()

	pr, pw := io.Pipe()
	tailErr := make(chan error, 1)
	go func() {
		tailErr <- tailFile(ctx, cmd.TmpStat, ffmpegDone, pw)
	}()

	var sum *ffmpeg.Summary
	var parseErr error
	switch kind {
	case ffmpeg.MetricPSNR:
		sum, parseErr = ffmpeg.ParsePSNR(pr, emitFrame)
	case ffmpeg.MetricSSIM:
		sum, parseErr = ffmpeg.ParseSSIM(pr, emitFrame)
	case ffmpeg.MetricXPSNR:
		sum, parseErr = ffmpeg.ParseXPSNR(pr, emitFrame)
	case ffmpeg.MetricVMAF:
		sum, parseErr = ffmpeg.ParseVMAFCSV(pr, emitFrame)
	}
	// Drain pipe close + collect tail / ffmpeg results.
	_ = pr.Close()
	if tErr := <-tailErr; tErr != nil && parseErr == nil {
		parseErr = tErr
	}
	<-ffmpegDone
	if ffmpegErr != nil {
		var exitErr *exec.ExitError
		if errors.As(ffmpegErr, &exitErr) {
			return fmt.Errorf("ffmpeg failed for %s/%s: %v", filepath.Base(dist), kind, ffmpegErr)
		}
		return ffmpegErr
	}
	if parseErr != nil {
		return parseErr
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

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// tailFile follows path while ffmpeg is running: it polls for the file's
// existence, streams any newly appended bytes into w, and after ffmpegDone
// fires it performs one last read to drain whatever was flushed at exit
// (this is how we capture libvmaf's end-of-run dump). w is closed on return.
func tailFile(ctx context.Context, path string, ffmpegDone <-chan struct{}, w *io.PipeWriter) error {
	defer w.Close()
	var f *os.File
	defer func() {
		if f != nil {
			f.Close()
		}
	}()
	openIfNeeded := func() {
		if f != nil {
			return
		}
		if of, err := os.Open(path); err == nil {
			f = of
		}
	}
	copyAvailable := func() error {
		if f == nil {
			return nil
		}
		buf := make([]byte, 32*1024)
		for {
			n, err := f.Read(buf)
			if n > 0 {
				if _, werr := w.Write(buf[:n]); werr != nil {
					return werr
				}
			}
			if err == io.EOF || n == 0 {
				return nil
			}
			if err != nil {
				return err
			}
		}
	}
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ffmpegDone:
			// final drain: file may have been written/flushed just now
			openIfNeeded()
			return copyAvailable()
		case <-ticker.C:
			openIfNeeded()
			if err := copyAvailable(); err != nil {
				return err
			}
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
