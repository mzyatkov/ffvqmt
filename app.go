package main

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/mzyatkov/ffvqmt/internal/cli"
	"github.com/mzyatkov/ffvqmt/internal/csvlog"
	"github.com/mzyatkov/ffvqmt/internal/ffmpeg"
	"github.com/mzyatkov/ffvqmt/internal/metrics"
	"github.com/mzyatkov/ffvqmt/internal/project"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App is the Wails-bound application object exposing all backend methods to JS.
type App struct {
	ctx       context.Context
	cliOpts   *cli.Options
	prober    *ffmpeg.Prober
	probeInfo *ffmpeg.ProbeInfo
	runMu     sync.Mutex
	cancelFn  context.CancelFunc
}

// NewApp constructs the application object.
func NewApp(opts *cli.Options) *App {
	return &App{
		cliOpts: opts,
		prober:  ffmpeg.NewProber(opts.FFmpegDir),
	}
}

// Startup is called by Wails when the app is launched.
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
}

// DomReady fires after the frontend has mounted.
func (a *App) DomReady(ctx context.Context) {
	// Push initial config to UI
	wailsruntime.EventsEmit(ctx, "cli:options", a.cliOpts)
	if a.cliOpts.Run {
		wailsruntime.EventsEmit(ctx, "cli:autorun", true)
	}
}

// Shutdown is called when the app exits.
func (a *App) Shutdown(ctx context.Context) {
	if a.cancelFn != nil {
		a.cancelFn()
	}
}

// ===== Methods exposed to the frontend =====

// GetInitialOptions returns CLI-derived options.
func (a *App) GetInitialOptions() *cli.Options { return a.cliOpts }

// DetectFFmpeg probes ffmpeg/ffprobe availability and supported filters.
func (a *App) DetectFFmpeg() (*ffmpeg.ProbeInfo, error) {
	info, err := a.prober.Probe()
	if err != nil {
		return nil, err
	}
	a.probeInfo = info
	return info, nil
}

// MediaInfo returns ffprobe details for a single file.
func (a *App) MediaInfo(path string) (*ffmpeg.MediaInfo, error) {
	return a.prober.MediaInfo(path)
}

// MakeThumbnail extracts a representative frame to PNG and returns its path.
func (a *App) MakeThumbnail(path string) (string, error) {
	return a.prober.Thumbnail(path)
}

// SelectFiles opens a native file picker; returns selected paths.
func (a *App) SelectFiles(multi bool, title string) ([]string, error) {
	if multi {
		return wailsruntime.OpenMultipleFilesDialog(a.ctx, wailsruntime.OpenDialogOptions{
			Title: title,
			Filters: []wailsruntime.FileFilter{
				{DisplayName: "Video files", Pattern: "*.mp4;*.mkv;*.avi;*.mov;*.webm;*.y4m;*.yuv;*.avs;*.ts"},
				{DisplayName: "All files", Pattern: "*.*"},
			},
		})
	}
	one, err := wailsruntime.OpenFileDialog(a.ctx, wailsruntime.OpenDialogOptions{Title: title})
	if err != nil || one == "" {
		return nil, err
	}
	return []string{one}, nil
}

// SaveFileDialog asks for a save path.
func (a *App) SaveFileDialog(title, defaultName string) (string, error) {
	return wailsruntime.SaveFileDialog(a.ctx, wailsruntime.SaveDialogOptions{
		Title:           title,
		DefaultFilename: defaultName,
	})
}

// SelectDirectory opens a native directory picker.
func (a *App) SelectDirectory(title string) (string, error) {
	return wailsruntime.OpenDirectoryDialog(a.ctx, wailsruntime.OpenDialogOptions{Title: title})
}

// LoadProject reads a .ffvqmtproj file.
func (a *App) LoadProject(path string) (*project.Project, error) {
	return project.Load(path)
}

// SaveProject writes a .ffvqmtproj file.
func (a *App) SaveProject(path string, p *project.Project) error {
	return project.Save(path, p)
}

// StartRun begins a calculation. It runs asynchronously and emits events:
//
//	metric:frame   { file, metric, frame, value }
//	metric:summary { file, metric, summary }
//	run:progress   { file, percent }
//	run:done       { ok, error }
func (a *App) StartRun(req metrics.RunRequest) error {
	a.runMu.Lock()
	defer a.runMu.Unlock()

	if a.probeInfo == nil {
		info, err := a.prober.Probe()
		if err != nil {
			return fmt.Errorf("ffmpeg detection failed: %w", err)
		}
		a.probeInfo = info
	}

	ctx, cancel := context.WithCancel(a.ctx)
	a.cancelFn = cancel

	runner := metrics.NewRunner(a.prober, a.probeInfo, a.emit)
	go func() {
		defer cancel()
		err := runner.Run(ctx, req)
		ok := err == nil
		msg := ""
		if err != nil {
			msg = err.Error()
		}
		a.emit("run:done", map[string]any{"ok": ok, "error": msg})

		if ok && req.AutoSaveResults {
			out := req.AutoSaveResultsFile
			if out == "" {
				out = filepath.Join(req.LogFramesDir, "FFvqmt.results.csv")
			}
			if err := csvlog.AppendSummary(out, runner.Summaries()); err != nil {
				a.emit("log:error", err.Error())
			}
		}
	}()
	return nil
}

// CancelRun aborts the current run if any.
func (a *App) CancelRun() {
	if a.cancelFn != nil {
		a.cancelFn()
	}
}

func (a *App) emit(event string, payload any) {
	if a.ctx != nil {
		wailsruntime.EventsEmit(a.ctx, event, payload)
	}
}
