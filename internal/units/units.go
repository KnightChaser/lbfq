// internal/units/units.go
package units

import (
	"fmt"
	"strconv"
	"strings"
)

// Converts a human-readable size string (e.g., "10K", "5M", "2G") into its equivalent size in bytes.
func ParseSize(s string) (int64, error) {
	s = strings.TrimSpace(strings.ToUpper(s))
	if s == "" {
		return 0, nil
	}

	m := int64(1)
	switch {
	case strings.HasSuffix(s, "K"):
		m, s = 1<<10, strings.TrimSuffix(s, "K")
	case strings.HasSuffix(s, "M"):
		m, s = 1<<20, strings.TrimSuffix(s, "M")
	case strings.HasSuffix(s, "G"):
		m, s = 1<<30, strings.TrimSuffix(s, "G")
	case strings.HasSuffix(s, "T"):
		m, s = 1<<40, strings.TrimSuffix(s, "T")
	}

	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, err
	}

	return int64(f * float64(m)), nil
}

// Converts a size in bytes into a human-readable string (e.g., 1536 -> "1.5KiB").
// We stick to "GiB" style instead of "GB" to avoid ambiguity. (2^X instead of 10^Y)
func Human(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%dB", n)
	}

	div, exp := int64(unit), 0
	for v := n / unit; v >= unit; v /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f%ciB", float64(n)/float64(div), "KMGTPE"[exp])
}
