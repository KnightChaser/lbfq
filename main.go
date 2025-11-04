package main

import (
	"container/heap"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
)

type fileRec struct {
	size int64
	path string
}

/* ---------- tiny min-heap to keep top-N ---------- */
type minHeap []fileRec

func (h minHeap) Len() int            { return len(h) }
func (h minHeap) Less(i, j int) bool  { return h[i].size < h[j].size }
func (h minHeap) Swap(i, j int)       { h[i], h[j] = h[j], h[i] }
func (h *minHeap) Push(x interface{}) { *h = append(*h, x.(fileRec)) }
func (h *minHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[:n-1]
	return x
}

/* ----------- helpers ----------------------------- */
func parseSize(s string) (int64, error) {
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

func human(n int64) string {
	/*
	 Convert a size in bytes to a human-readable string.
	 e.g., 1536 -> "1.5KiB"
	*/
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

func fileBytes(info fs.FileInfo, apparent bool) int64 {
	/*
	 Return the size of the file in bytes.
	*/
	if apparent {
		return info.Size()
	}
	if st, ok := info.Sys().(*syscall.Stat_t); ok {
		return st.Blocks * 512 // POSIX: st_blocks in 512B units
	}
	return info.Size()
}

func devOf(path string) (uint64, error) {
	/*
	 Return the device ID of the filesystem containing the given path.
	*/
	var st syscall.Stat_t
	if err := syscall.Lstat(path, &st); err != nil {
		return 0, err
	}
	return uint64(st.Dev), nil
}

func shouldSkip(path string, skipPrefixes []string) bool {
	/*
	 Return true if the path starts with any of the skip prefixes.
	*/
	for _, p := range skipPrefixes {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}

/* ---------- main ---------- */
func main() {
	// Keep flags minimal for v0
	root := flag.String("root", "/", "directory to scan")
	topN := flag.Int("n", 50, "show top N largest files")
	minStr := flag.String("min", "0", "only list files >= size (e.g. 100M, 1G)")
	xdev := flag.Bool("xdev", true, "stay on the same filesystem")
	apparent := flag.Bool("apparent", false, "use apparent size instead of on-disk bytes")
	workers := flag.Int("workers", 8, "concurrent stat workers")
	flag.Parse()

	minSize, err := parseSize(*minStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid -min: %v\n", err)
		os.Exit(2)
	}

	// Identify root device for -xdev
	var rootDev uint64
	if *xdev {
		if d, err := devOf(*root); err == nil {
			rootDev = d
		} else {
			fmt.Fprintf(os.Stderr, "stat root: %v\n", err)
			os.Exit(2)
		}
	}

	// NOTE:
	// Hard-coded safe skips for Linux.
	// Generally, they are not a target for disk usage optimization.
	skips := []string{"/proc", "/sys", "/run", "/dev"}

	paths := make(chan string, 4096)
	type res struct {
		size int64
		path string
	}
	results := make(chan res, 4096)

	var wg sync.WaitGroup

	// Producer: WalkDir pushes candidate paths
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = filepath.WalkDir(*root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				// unreadable -> skip
				return nil
			}

			// quick exclusions
			if shouldSkip(path, skips) {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			// stay on same filesystem: check device for dirs (prune) and files (drop)
			if *xdev {
				var st syscall.Stat_t
				if err := syscall.Lstat(path, &st); err == nil {
					if uint64(st.Dev) != rootDev {
						if d.IsDir() {
							return filepath.SkipDir
						}
						return nil
					}
				}
			}

			// ignore symlinks entirely
			if d.Type()&os.ModeSymlink != 0 {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			paths <- path
			return nil
		})
		close(paths)
	}()

	// Consumers: stat files and emit results
	for i := 0; i < *workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for p := range paths {
				info, err := os.Lstat(p)
				if err != nil || info.IsDir() {
					continue
				}
				sz := fileBytes(info, *apparent)
				if sz >= minSize {
					results <- res{size: sz, path: p}
				}
			}
		}()
	}

	// Close results when workers finish
	go func() {
		wg.Wait()
		close(results)
	}()

	// Keep top-N in a min-heap
	h := &minHeap{}
	heap.Init(h)
	for r := range results {
		if h.Len() < *topN {
			// Fill up heap first
			heap.Push(h, fileRec{size: r.size, path: r.path})
			continue
		}
		if (*h)[0].size < r.size {
			heap.Pop(h)
			heap.Push(h, fileRec{size: r.size, path: r.path})
		}
	}

	// Drain heap to slice (ascending), then print descending
	out := make([]fileRec, h.Len())
	for i := len(out) - 1; i >= 0; i-- {
		out[i] = heap.Pop(h).(fileRec)
	}
	for i := len(out) - 1; i >= 0; i-- {
		fmt.Printf("%12s  %s\n", human(out[i].size), out[i].path)
	}
}
