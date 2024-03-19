package resources

import (
	"cmp"
	"fmt"
	"strconv"
	"strings"
)

type version struct {
	major int
	minor int
	patch int
}

func versionFromDirectoryName(name string) (version, error) {
	if name == "vanilla" {
		return version{}, nil
	}

	if !strings.HasPrefix(name, "vanilla_") {
		return version{}, fmt.Errorf("unexpected resource pack name. Want vanilla_(number).(number)[.(number)]; got %q", name)
	}

	v, err := newVersion(name[8:])
	if err != nil {
		return version{}, fmt.Errorf("error parsing version number out of resource pack named %q", name)
	}
	return v, nil
}

func newVersion(s string) (version, error) {
	var v version
	var err error

	parts := strings.Split(s, ".")
	if len(parts) > 3 {
		return v, fmt.Errorf("weird version number has >3 dotted parts: %q", s)
	}

	v.major, err = strconv.Atoi(parts[0])
	if err != nil {
		return v, fmt.Errorf("error parsing first part of version %q", s)
	}

	if len(parts) > 1 {
		v.minor, err = strconv.Atoi(parts[1])
		if err != nil {
			return v, fmt.Errorf("error parsing second part of version %q", s)
		}
		if len(parts) > 2 {
			v.patch, err = strconv.Atoi(parts[2])
			if err != nil {
				return v, fmt.Errorf("error parsing third part of version %q", s)
			}
		}
	}

	return v, nil
}

func (a version) cmp(b version) int {
	if a.major != b.major {
		return cmp.Compare(a.major, b.major)
	}
	if a.minor != b.minor {
		return cmp.Compare(a.minor, b.minor)
	}
	if a.patch != b.patch {
		return cmp.Compare(a.patch, b.patch)
	}
	return 0
}
