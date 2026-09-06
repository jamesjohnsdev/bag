// Package output holds small presentation helpers shared between cmd (which
// prints phase headers) and provider (which streams download progress) - kept
// separate from both so neither has to import the other just for this.
package output

import (
	"fmt"
	"io"

	"github.com/fatih/color"
	"github.com/schollz/progressbar/v3"
)

// Statusf prints a pacman-style "::" status line for a phase of an install/update.
func Statusf(format string, a ...any) {
	prefix := color.New(color.FgBlue, color.Bold).Sprint("::")
	fmt.Printf("%s %s\n", prefix, fmt.Sprintf(format, a...))
}

// HumanSize formats a byte count as a human-readable string (KiB/MiB/GiB),
// or "unknown size" when size is negative (content length wasn't available).
func HumanSize(size int64) string {
	if size < 0 {
		return "unknown size"
	}
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(size)/float64(div), "KMGTPE"[exp])
}

// WrapProgress wraps rc in a live progress bar keyed on size (-1 if unknown),
// labelled with label. Skipped when output isn't a terminal, matching the
// rest of the CLI's color.NoColor gating.
func WrapProgress(rc io.ReadCloser, size int64, label string) io.ReadCloser {
	if color.NoColor {
		// bail out for non-tty/pipe
		return rc
	}
	bar := progressbar.DefaultBytes(size, label)
	wrappedReader := progressbar.NewReader(rc, bar)
	return &wrappedReader
}
