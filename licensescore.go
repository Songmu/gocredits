package gocredits

import "regexp"

// copied from github.com/pmezard/licenses
var (
	reLicense = regexp.MustCompile(`(?i)^(?:` +
		`((?:un)?licen[sc]e)|` +
		`((?:un)?licen[sc]e\.(?:md|markdown|txt))|` +
		`(copy(?:ing|right)(?:\.[^.]+)?)|` +
		`(licen[sc]e\.[^.]+)` +
		`)$`)
)

// scoreLicenseName returns a factor between 0 and 1 weighting how likely
// supplied filename is a license file.
func scoreLicenseName(name string) float64 {
	m := reLicense.FindStringSubmatch(name)
	switch {
	case m == nil:
		break
	case m[1] != "":
		return 1.0
	case m[2] != "":
		return 0.9
	case m[3] != "":
		return 0.8
	case m[4] != "":
		return 0.7
	}
	return 0.
}
