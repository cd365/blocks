package program

const (
	ModeDebug   = "DEBUG"
	ModeTest    = "TEST"
	ModeRelease = "RELEASE"
)

var (
	// BuildAt program build time. "2006-01-02 15:04:05"
	BuildAt string

	// RunMode program running mode.
	RunMode string

	// VersionControlId program version control id.
	VersionControlId string

	// Version program version.
	Version = "v0.0.0"
)

func IsDebugMode() bool {
	return RunMode == ModeDebug || RunMode == ""
}

func IsTestMode() bool {
	return RunMode == ModeTest
}

func IsReleaseMode() bool {
	return RunMode == ModeRelease
}
