package utils

import "strings"

func EscapeUnderscores(s string) string {
	return strings.ReplaceAll(s, "_", "\\_")
}
