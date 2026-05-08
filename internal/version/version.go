package version

// These variables are set at link time via -ldflags.
// Defaults make local dev builds identifiable without a tag.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)
