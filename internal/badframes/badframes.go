// Package badframes extracts "bad" frames as PNG images using ffmpeg.
package badframes

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
)

// Extract extracts the given frame numbers (1-based) from `path` to dir/prefix-N.png.
// fps is required for ffmpeg seeking.
func Extract(ctx context.Context, ffmpegBin, path, outDir, prefix string, fps float64, frames []int) error {
	if fps <= 0 {
		fps = 25
	}
	for _, n := range frames {
		ts := float64(n) / fps
		out := filepath.Join(outDir, fmt.Sprintf("%s-%06d.png", prefix, n))
		args := []string{
			"-hide_banner", "-loglevel", "error",
			"-ss", strconv.FormatFloat(ts, 'f', 3, 64),
			"-i", path,
			"-frames:v", "1",
			"-y", out,
		}
		cmd := exec.CommandContext(ctx, ffmpegBin, args...)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("extract frame %d: %w", n, err)
		}
	}
	return nil
}
