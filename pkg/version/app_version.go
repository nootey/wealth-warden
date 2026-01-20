package version

var (
	// Version info (injected at build time via ldflags)
	Version   = "dev"
	CommitSHA = "unknown"
	BuildTime = "unknown"
)
