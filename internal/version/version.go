package version

import "fmt"

var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

func String() string {
	return fmt.Sprintf("nexus %s (commit: %s, built: %s)", Version, Commit, BuildDate)
}

func Short() string {
	return Version
}
