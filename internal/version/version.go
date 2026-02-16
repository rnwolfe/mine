package version

// Set at build time via ldflags. See Makefile for details.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

func Full() string {
	return Version + " (" + Commit + ") " + Date
}

func Short() string {
	return Version
}
