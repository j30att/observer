package buildinfo

import (
	"fmt"
	"io"
)

// Print writes build metadata to writer.
func Print(writer io.Writer, version, date, commit string) {
	fmt.Fprintf(writer, "Build version: %s\n", value(version))
	fmt.Fprintf(writer, "Build date: %s\n", value(date))
	fmt.Fprintf(writer, "Build commit: %s\n", value(commit))
}

func value(value string) string {
	if value == "" {
		return "N/A"
	}

	return value
}
