// internal/scan/workers.go
package scan

import (
	"runtime"
)

// Calculate an automatic number of workers based on CPU cores
func autoWorkers() int {
	n := runtime.NumCPU()
	if n < 1 {
		n = 1
	}

	// Considering IO-bound, we expect a performance benefit
	// by oversubscribing workers by a factor of 2.
	n *= 2
	if n < 4 {
		n = 4
	}
	if n > 64 {
		n = 64
	}
	return n
}
