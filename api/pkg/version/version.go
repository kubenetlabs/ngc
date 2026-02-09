package version

// Version, Commit, and Date are set via ldflags at build time.
//
//	go build -ldflags "-X github.com/kubenetlabs/ngc/api/pkg/version.Version=v1.0.0
//	  -X github.com/kubenetlabs/ngc/api/pkg/version.Commit=abc1234
//	  -X github.com/kubenetlabs/ngc/api/pkg/version.Date=2025-01-01T00:00:00Z"
var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)
