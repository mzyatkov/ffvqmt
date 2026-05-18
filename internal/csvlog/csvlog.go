// Package csvlog writes tab-delimited CSVs for FFvqmt aggregate results.
package csvlog

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/mzyatkov/ffvqmt/internal/ffmpeg"
	"github.com/mzyatkov/ffvqmt/internal/metrics"
)

// AppendSummary appends one row per (file, metric) to the given CSV path.
// Creates the file with a header row if missing.
func AppendSummary(path string, summaries map[metrics.SummaryKey]*ffmpeg.Summary) error {
	if path == "" {
		return fmt.Errorf("empty results path")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	needsHeader := false
	if st, err := os.Stat(path); err != nil || st.Size() == 0 {
		needsHeader = true
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	if needsHeader {
		fmt.Fprintln(f, "datetime\tfile\tmetric\tframes\tmin\tmax\taverage\tharmonic_mean")
	}
	keys := make([]metrics.SummaryKey, 0, len(summaries))
	for k := range summaries {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].File != keys[j].File {
			return keys[i].File < keys[j].File
		}
		return keys[i].Metric < keys[j].Metric
	})
	now := time.Now().Format(time.RFC3339)
	for _, k := range keys {
		s := summaries[k]
		fmt.Fprintf(f, "%s\t%s\t%s\t%d\t%.6f\t%.6f\t%.6f\t%.6f\n",
			now,
			escapeTab(k.File),
			k.Metric,
			s.Frames,
			s.Min,
			s.Max,
			s.Average,
			s.Harmonic,
		)
	}
	return nil
}

func escapeTab(s string) string {
	return strings.ReplaceAll(s, "\t", " ")
}
