// internal/scan/workers.go
package scan

import (
	"runtime"
)

// Calculate an automatic number of workers based on CPU cores
func autoWorkers() int {
	n := max(1, runtime.NumCPU())
	n *= 2
	n = max(4, n)
	n = min(64, n)
	return n
}
