package common

var (
	Version   = "dev"
	Build     = "unknown"
	GitCommit = "unknown"
)

func GetVersion() string {
	return Version
}

func GetBuild() string {
	return Build
}

func GetGitCommit() string {
	return GitCommit
}
