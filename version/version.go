package version

var (
	Version string = IPOSSemVer

	GitCommit string

	UserAgent string
)

func init() {
	UserAgent = "ipos/" + Version
	if GitCommit != "" {
		Version += "-" + GitCommit
		UserAgent += "/" + GitCommit
	}
}

const (
	IPOSSemVer = "0.1.0"
)
