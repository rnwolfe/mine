package version

import "runtime/debug"

// Set at build time via ldflags.
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

// init backfills Version, Commit, and Date from runtime/debug build info when
// the ldflags defaults are still in place. ldflags values always take precedence.
func init() {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}
	backfillFromBuildInfo(info)
}

// backfillFromBuildInfo fills in Version, Commit, and Date from the provided
// BuildInfo when the respective variable still holds its ldflag default value.
// This allows `go install` builds to show real version info without requiring
// ldflags to be passed.
func backfillFromBuildInfo(info *debug.BuildInfo) {
	if info == nil {
		return
	}

	// Only set Version if still at default and build info has a real tagged version.
	// When built from HEAD without a tag, info.Main.Version == "(devel)" — keep "dev".
	if Version == "dev" && info.Main.Version != "" && info.Main.Version != "(devel)" {
		Version = info.Main.Version
	}

	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			if Commit == "none" && s.Value != "" {
				rev := s.Value
				if len(rev) > 7 {
					rev = rev[:7]
				}
				Commit = rev
			}
		case "vcs.time":
			if Date == "unknown" && s.Value != "" {
				Date = s.Value
			}
		}
	}
}
