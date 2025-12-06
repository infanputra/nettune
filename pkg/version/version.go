// Package version provides build information
package version

import (
	"fmt"
	"runtime"
)

var (
	// Version is the semantic version (set by ldflags)
	Version = "dev"

	// GitCommit is the git commit hash (set by ldflags)
	GitCommit = "unknown"

	// BuildDate is the build timestamp (set by ldflags)
	BuildDate = "unknown"
)

// Info represents build version information
type Info struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
	BuildDate string `json:"build_date"`
	GoVersion string `json:"go_version"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
}

// GetInfo returns the current version information
func GetInfo() Info {
	return Info{
		Version:   Version,
		GitCommit: GitCommit,
		BuildDate: BuildDate,
		GoVersion: runtime.Version(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
	}
}

// String returns a formatted version string
func (i Info) String() string {
	return fmt.Sprintf("nettune %s (%s) built on %s with %s",
		i.Version, i.GitCommit, i.BuildDate, i.GoVersion)
}
