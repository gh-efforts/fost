package build

var CurrentCommit string

const Version = "0.1.0"

func UserVersion() string {
	return Version + CurrentCommit
}
