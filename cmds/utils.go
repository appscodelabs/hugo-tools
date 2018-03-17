package cmds

import "strings"

func clean(s string) string {
	s = strings.Replace(s, "_", " ", -1)
	s = strings.Replace(s, "-", " ", -1)
	return s
}

func id(s string) string {
	return strings.ToLower(strings.Replace(s, " ", "-", -1))
}
