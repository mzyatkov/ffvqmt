// Package project handles loading/saving .ffvqmtproj files (JSON).
package project

import (
	"encoding/json"
	"os"
)

// Project holds the persistable state of an FFvqmt session.
type Project struct {
	Version       int            `json:"version"`
	RefFile       string         `json:"refFile"`
	DistFiles     []DistEntry    `json:"distFiles"`
	Metrics       []string       `json:"metrics"`
	Skip          float64        `json:"skip"`
	Duration      float64        `json:"duration"`
	Scaling       string         `json:"scaling"`
	VMAFModel     string         `json:"vmafModel"`
	VMAFPool      string         `json:"vmafPool"`
	VMAFSubsample int            `json:"vmafSubsample"`
	VMAFPhone     bool           `json:"vmafPhone"`
	VMAFUpscale   bool           `json:"vmafUpscale"`
	Extra         map[string]any `json:"extra,omitempty"`
}

// DistEntry is a distorted file with its active flag.
type DistEntry struct {
	Path   string `json:"path"`
	Active bool   `json:"active"`
}

// Load reads a project file.
func Load(path string) (*Project, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var p Project
	if err := json.Unmarshal(b, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// Save writes a project file (pretty-printed).
func Save(path string, p *Project) error {
	if p.Version == 0 {
		p.Version = 1
	}
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}
