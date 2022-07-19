package version

import "fmt"

const (
	Version           = "0.13.0"
	VersionPrerelease = "" // "-dev", "-beta", "-rc1", etc. (include dash)
)

var (
	Name      string = "envconsul"
	GitCommit string

	HumanVersion = fmt.Sprintf("%s v%s%s (%s)",
		Name, Version, VersionPrerelease, GitCommit)
)
