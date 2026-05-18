package cli

import (
	"reflect"
	"testing"
)

func TestParse_DefaultMetrics(t *testing.T) {
	o, err := Parse([]string{"ref.mp4", "a.mp4", "b.mp4"})
	if err != nil {
		t.Fatal(err)
	}
	if o.RefFile != "ref.mp4" {
		t.Errorf("ref: %q", o.RefFile)
	}
	if !reflect.DeepEqual(o.DistFiles, []string{"a.mp4", "b.mp4"}) {
		t.Errorf("dist: %v", o.DistFiles)
	}
	if !reflect.DeepEqual(o.Metrics, []string{"PSNR", "SSIM", "VMAF"}) {
		t.Errorf("metrics: %v", o.Metrics)
	}
}

func TestParse_FlagsAndMetrics(t *testing.T) {
	o, err := Parse([]string{
		"-log-frames",
		"--metric=SSIM",
		"-metric=VMAF",
		"--vmaf-pool=harmonic_mean",
		"-vmaf-subsample=2",
		"-run",
		"ref.mp4", "x.mp4",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !o.LogFrames || !o.Run {
		t.Error("bool flags not set")
	}
	if !reflect.DeepEqual(o.Metrics, []string{"SSIM", "VMAF"}) {
		t.Errorf("metrics: %v", o.Metrics)
	}
	if string(o.VMAFPool) != "HARMONIC_MEAN" {
		t.Errorf("vmaf-pool: %v", o.VMAFPool)
	}
	if o.VMAFSubsample != 2 {
		t.Errorf("vmaf-subsample: %v", o.VMAFSubsample)
	}
}

func TestParse_UpscaleBool(t *testing.T) {
	o, _ := Parse([]string{"-vmaf-upscale-to-model=false"})
	if o.VMAFUpscaleToModel {
		t.Error("expected false")
	}
	o, _ = Parse([]string{"-vmaf-upscale-to-model"})
	if !o.VMAFUpscaleToModel {
		t.Error("expected true (default presence)")
	}
}
