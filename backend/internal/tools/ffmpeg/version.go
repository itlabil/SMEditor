package ffmpeg

import "strings"

// parseVersion extracts the token after the literal "version" in output
// like "ffmpeg version 6.1.1-3ubuntu5 Copyright (c) 2000-2023 ...".
func parseVersion(output string) string {
	fields := strings.Fields(output)
	for i, f := range fields {
		if f == "version" && i+1 < len(fields) {
			return fields[i+1]
		}
	}
	return ""
}
