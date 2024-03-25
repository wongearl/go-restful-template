package version

import "fmt"

const (
	// CommercialEdition is the default edition for building.
	CommercialEdition = "Commercial"
)

var (
	releaseVersion = "None"
	buildDate      = "None"
	gitHash        = "None"
	gitBranch      = "None"
	edition        = CommercialEdition
)

type VersionInfo struct {
	ReleaseVersion string `json:"releaseVersion"`
	Edition        string `json:"edition"`
	GitHash        string `json:"gitHash"`
	GitBranch      string `json:"gitBranch"`
	BuildDate      string `json:"buildDate"`
}

// String returns the formatted version string
func (v VersionInfo) String() string {
	return fmt.Sprintf(`Version: %s
Edition: %s
GitHash: %s
GitBranch: %s
BuildDate: %s`, v.ReleaseVersion, v.Edition, v.GitHash, v.GitBranch, v.BuildDate)
}

// Get resturns the overall codebase version. It's for
// detecting what code a binary was built from.
func Get() VersionInfo {
	return VersionInfo{
		ReleaseVersion: releaseVersion,
		Edition:        edition,
		GitHash:        gitHash,
		GitBranch:      gitBranch,
		BuildDate:      buildDate,
	}
}
