package ffmpeg

import (
	"strings"
	"testing"
)

func TestParsePSNR(t *testing.T) {
	in := strings.NewReader(strings.Join([]string{
		"n:1 mse_avg:1.234 mse_y:1.0 mse_u:0.0 mse_v:0.0 psnr_avg:48.234 psnr_y:50 psnr_u:Inf psnr_v:Inf",
		"n:2 mse_avg:2.000 mse_y:2.0 mse_u:0.0 mse_v:0.0 psnr_avg:45.111 psnr_y:46 psnr_u:Inf psnr_v:Inf",
		"",
	}, "\n"))
	got := 0
	s, err := ParsePSNR(in, func(fs FrameSample) { got++ })
	if err != nil {
		t.Fatal(err)
	}
	if got != 2 || s.Frames != 2 {
		t.Fatalf("frames: %d got:%d", s.Frames, got)
	}
	if s.Min > s.Max {
		t.Error("min > max")
	}
}

func TestParseSSIM(t *testing.T) {
	in := strings.NewReader("n:1 Y:0.99 U:0.98 V:0.97 All:0.98 (17.99)\nn:2 Y:0.97 U:0.95 V:0.95 All:0.96 (12.0)\n")
	s, err := ParseSSIM(in, nil)
	if err != nil {
		t.Fatal(err)
	}
	if s.Frames != 2 {
		t.Fatalf("frames: %d", s.Frames)
	}
	if s.Min != 0.96 {
		t.Errorf("min: %f", s.Min)
	}
	if s.Max != 0.98 {
		t.Errorf("max: %f", s.Max)
	}
}

func TestParseVMAFCSV(t *testing.T) {
	in := strings.NewReader("Frame,vmaf\n0,98.5\n1,97.0\n2,99.0\n")
	s, err := ParseVMAFCSV(in, nil)
	if err != nil {
		t.Fatal(err)
	}
	if s.Frames != 3 || s.Max != 99.0 {
		t.Fatalf("got: %+v", s)
	}
}
