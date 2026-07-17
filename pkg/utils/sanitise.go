package utils

import "strings"

// Device alias can't have any slashes or
// other invalid characters for a device path.
func SanitiseAliasName(input string) string {
	// Lowercase alphanumeric characters and dash.
	valid := "abcdefghijklmnopqrstuvwxyz1234567890-"

	var out strings.Builder
	for _, char := range strings.Split(input, "") {
		c := strings.ToLower(char)
		if strings.Contains(valid, c) {
			out.WriteString(c)
		} else {
			// Replace invalid with dash.
			out.WriteString("-")
		}
	}

	return out.String()
}

// Make input string into a valid environment variable name.
// Replace non alphanumeric characters with underscores.
func SanitiseEnvName(input string) string {
	// Alphanumeric characters and underscore.
	valid := "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_"

	var out strings.Builder
	for _, char := range strings.Split(input, "") {
		c := strings.ToUpper(char)
		if strings.Contains(valid, c) {
			out.WriteString(c)
		} else {
			out.WriteString("_")
		}
	}

	return out.String()
}
