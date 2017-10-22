package version

// nolint
const (
	Maj = "0"
	Min = "1"
	Fix = "0"
)

var (
	// Version is the full version string
	Version = "0.1.0"

	// GitCommit is set with --ldflags "-X main.gitCommit=$(git rev-parse HEAD)"
	GitCommit string
)

func init() {
	if GitCommit != "" {
		Version += "-" + GitCommit[:8]
	}
}
