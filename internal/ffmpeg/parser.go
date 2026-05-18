package ffmpeg

import (
	"bufio"
	"io"
	"regexp"
	"strconv"
	"strings"
)

// FrameSample is one parsed frame of metric output.
type FrameSample struct {
	Frame int     `json:"frame"`
	Value float64 `json:"value"`
	// Auxiliary per-frame values (mse_avg etc.) keyed by name.
	Extra map[string]float64 `json:"extra,omitempty"`
}

// Summary holds aggregate metric statistics from a complete run.
type Summary struct {
	Min      float64            `json:"min"`
	Max      float64            `json:"max"`
	Average  float64            `json:"average"`
	Harmonic float64            `json:"harmonic"`
	Frames   int                `json:"frames"`
	Extra    map[string]float64 `json:"extra,omitempty"`
}

// Reset Summary helpers
func emptySummary() *Summary { return &Summary{Min: 0, Max: 0, Extra: map[string]float64{}} }

// ParsePSNR parses a PSNR stats file: lines look like
//
//	n:1 mse_avg:1.234 mse_y:1.0 mse_u:0.0 mse_v:0.0 psnr_avg:48.234 psnr_y:50 psnr_u:Inf psnr_v:Inf
func ParsePSNR(r io.Reader, onFrame func(FrameSample)) (*Summary, error) {
	return parseKVStats(r, "psnr_avg", []string{"psnr_y", "psnr_u", "psnr_v", "mse_avg"}, onFrame)
}

// ParseSSIM parses SSIM stats: n:1 Y:0.99 U:0.99 V:0.99 All:0.99 (0.123)
func ParseSSIM(r io.Reader, onFrame func(FrameSample)) (*Summary, error) {
	s := emptySummary()
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	re := regexp.MustCompile(`(\w+):([-+]?[\d.]+|Inf|inf|nan)`)
	for scanner.Scan() {
		line := scanner.Text()
		kv := map[string]float64{}
		for _, m := range re.FindAllStringSubmatch(line, -1) {
			v := parseFloat(m[2])
			kv[m[1]] = v
		}
		nf, ok := kv["n"]
		if !ok {
			continue
		}
		val, ok := kv["All"]
		if !ok {
			val = kv["all"]
		}
		fs := FrameSample{Frame: int(nf), Value: val, Extra: kv}
		if onFrame != nil {
			onFrame(fs)
		}
		s.Frames++
		if s.Frames == 1 || val < s.Min {
			s.Min = val
		}
		if val > s.Max {
			s.Max = val
		}
		s.Average += val
		if val > 0 {
			s.Harmonic += 1.0 / val
		}
	}
	if s.Frames > 0 {
		s.Average /= float64(s.Frames)
		if s.Harmonic > 0 {
			s.Harmonic = float64(s.Frames) / s.Harmonic
		}
	}
	return s, scanner.Err()
}

// ParseXPSNR parses XPSNR stats; ffmpeg writes lines like:
//
//	n: 1 XPSNR y: 45.23 XPSNR u: 47.10 XPSNR v: 47.05 XPSNR yuv: 45.78
func ParseXPSNR(r io.Reader, onFrame func(FrameSample)) (*Summary, error) {
	s := emptySummary()
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	reN := regexp.MustCompile(`n\s*:\s*(\d+)`)
	reYUV := regexp.MustCompile(`XPSNR\s*yuv\s*:\s*([-+]?[\d.]+|inf|Inf|nan)`)
	reY := regexp.MustCompile(`XPSNR\s*y\s*:\s*([-+]?[\d.]+|inf|Inf|nan)`)
	reU := regexp.MustCompile(`XPSNR\s*u\s*:\s*([-+]?[\d.]+|inf|Inf|nan)`)
	reV := regexp.MustCompile(`XPSNR\s*v\s*:\s*([-+]?[\d.]+|inf|Inf|nan)`)
	for scanner.Scan() {
		line := scanner.Text()
		nm := reN.FindStringSubmatch(line)
		if nm == nil {
			continue
		}
		fn, _ := strconv.Atoi(nm[1])
		extra := map[string]float64{}
		if m := reYUV.FindStringSubmatch(line); m != nil {
			extra["yuv"] = parseFloat(m[1])
		}
		if m := reY.FindStringSubmatch(line); m != nil {
			extra["y"] = parseFloat(m[1])
		}
		if m := reU.FindStringSubmatch(line); m != nil {
			extra["u"] = parseFloat(m[1])
		}
		if m := reV.FindStringSubmatch(line); m != nil {
			extra["v"] = parseFloat(m[1])
		}
		val := extra["yuv"]
		if val == 0 {
			val = extra["y"]
		}
		fs := FrameSample{Frame: fn, Value: val, Extra: extra}
		if onFrame != nil {
			onFrame(fs)
		}
		s.Frames++
		if s.Frames == 1 || val < s.Min {
			s.Min = val
		}
		if val > s.Max {
			s.Max = val
		}
		s.Average += val
		if val > 0 {
			s.Harmonic += 1.0 / val
		}
	}
	if s.Frames > 0 {
		s.Average /= float64(s.Frames)
		if s.Harmonic > 0 {
			s.Harmonic = float64(s.Frames) / s.Harmonic
		}
	}
	return s, scanner.Err()
}

// ParseVMAFCSV parses libvmaf csv output (header + rows). The "vmaf" column is used.
func ParseVMAFCSV(r io.Reader, onFrame func(FrameSample)) (*Summary, error) {
	s := emptySummary()
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 256*1024), 4*1024*1024)
	var header []string
	first := true
	for scanner.Scan() {
		line := scanner.Text()
		if first {
			header = strings.Split(line, ",")
			first = false
			continue
		}
		cols := strings.Split(line, ",")
		if len(cols) != len(header) {
			continue
		}
		row := map[string]string{}
		for i, h := range header {
			row[strings.TrimSpace(h)] = strings.TrimSpace(cols[i])
		}
		fn, _ := strconv.Atoi(row["Frame"])
		if fn == 0 {
			fn, _ = strconv.Atoi(row["frame"])
		}
		val := parseFloat(row["vmaf"])
		if val == 0 {
			val = parseFloat(row["VMAF"])
		}
		extra := map[string]float64{}
		for k, v := range row {
			if k == "Frame" || k == "frame" {
				continue
			}
			if f := parseFloat(v); !isNaN(f) {
				extra[k] = f
			}
		}
		fs := FrameSample{Frame: fn, Value: val, Extra: extra}
		if onFrame != nil {
			onFrame(fs)
		}
		s.Frames++
		if s.Frames == 1 || val < s.Min {
			s.Min = val
		}
		if val > s.Max {
			s.Max = val
		}
		s.Average += val
		if val > 0 {
			s.Harmonic += 1.0 / val
		}
	}
	if s.Frames > 0 {
		s.Average /= float64(s.Frames)
		if s.Harmonic > 0 {
			s.Harmonic = float64(s.Frames) / s.Harmonic
		}
	}
	return s, scanner.Err()
}

func parseKVStats(r io.Reader, primary string, extras []string, onFrame func(FrameSample)) (*Summary, error) {
	s := emptySummary()
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	re := regexp.MustCompile(`(\w+):([-+]?[\d.]+|Inf|inf|nan)`)
	for scanner.Scan() {
		line := scanner.Text()
		kv := map[string]float64{}
		for _, m := range re.FindAllStringSubmatch(line, -1) {
			kv[m[1]] = parseFloat(m[2])
		}
		nf, ok := kv["n"]
		if !ok {
			continue
		}
		val := kv[primary]
		extra := map[string]float64{}
		for _, e := range extras {
			if v, ok := kv[e]; ok {
				extra[e] = v
			}
		}
		fs := FrameSample{Frame: int(nf), Value: val, Extra: extra}
		if onFrame != nil {
			onFrame(fs)
		}
		s.Frames++
		if s.Frames == 1 || val < s.Min {
			s.Min = val
		}
		if val > s.Max {
			s.Max = val
		}
		s.Average += val
		if val > 0 {
			s.Harmonic += 1.0 / val
		}
	}
	if s.Frames > 0 {
		s.Average /= float64(s.Frames)
		if s.Harmonic > 0 {
			s.Harmonic = float64(s.Frames) / s.Harmonic
		}
	}
	return s, scanner.Err()
}

func parseFloat(s string) float64 {
	s = strings.TrimSpace(s)
	switch strings.ToLower(s) {
	case "inf", "+inf":
		return 1e9
	case "-inf":
		return -1e9
	case "nan", "":
		return 0
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return v
}

func isNaN(f float64) bool { return f != f }
