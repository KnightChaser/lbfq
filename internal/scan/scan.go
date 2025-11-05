// internal/scan/scan.go
package scan

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
)

type Result struct {
	Size int64
	Path string
}

type Config struct {
	Root         string
	MinSize      int64
	XDev         bool
	Apparent     bool
	Workers      int      // 0 means auto-tune based on CPU cores
	Skips        []string // hard prefixes to skip (e.g. /proc)
	ExcludeGlobs []string // user globs matched on full path (e.g. *.log)
}

// Scan walks the tree and streams file results >= MinSize into the returned channel.
// The channel closes when scanning completes.
func Scan(cfg Config) <-chan Result {
	if cfg.Workers <= 0 {
		// NOTE: Assume there are enough I/O operations to keep 8 workers busy.
		cfg.Workers = autoWorkers()
	}
	paths := make(chan string, 4096)
	results := make(chan Result, 4096)

	var rootDev uint64
	if cfg.XDev {
		if d, err := devOf(cfg.Root); err == nil {
			rootDev = d
		} else {
			// leave rootDev=0 and skip xDev checks
			cfg.XDev = false
		}
	}

	var wg sync.WaitGroup

	// Producer: walk filesystem
	wg.Add(1)

	go func() {
		defer wg.Done()
		_ = filepath.WalkDir(cfg.Root, func(path string, d fs.DirEntry, err error) error {
			// NOTE: Skip unreadable directory
			if err != nil {
				return nil
			}

			// NOTE: Prune skipped prefixes and excluded globs
			if hasPrefix(path, cfg.Skips) {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			if matchAnyGlob(path, cfg.ExcludeGlobs) {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			// NOTE: XDev pruning (compare device for dirs and files)
			if cfg.XDev {
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

			// NOTE: Ignore symlinks entirely
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

	// Consumers: state files and emit >= MinSize
	for i := 0; i < cfg.Workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range paths {
				info, err := os.Lstat(path)
				if err != nil || info.IsDir() {
					continue
				}

				sz := fileBytes(info, cfg.Apparent)
				if sz >= cfg.MinSize {
					results <- Result{Size: sz, Path: path}
				}
			}
		}()
	}

	// Closer
	go func() {
		wg.Wait()
		close(results)
	}()

	return results
}

// Check if path matches any of the given globs
func matchAnyGlob(path string, globs []string) bool {
	if len(globs) == 0 {
		return false
	}

	p := filepath.Clean(path)
	for _, g := range globs {
		g = strings.TrimSpace(g)
		if g == "" {
			continue
		}

		// match against the full path
		if ok, _ := filepath.Match(g, p); ok {
			return true
		}
	}
	return false
}

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

// Check if path has any of the given prefixes
func hasPrefix(path string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}

// Get the size of the file, either apparent or on-disk
func fileBytes(info fs.FileInfo, apparent bool) int64 {
	if apparent {
		return info.Size()
	}
	if st, ok := info.Sys().(*syscall.Stat_t); ok {
		// NOTE:
		// POSIX's st_blocks is in 512-byte units -> on-disk bytes
		return st.Blocks * 512
	}
	return info.Size()
}

// Get the device ID of the filesystem containing the path
// It's for non-crossing device checks
func devOf(path string) (uint64, error) {
	var st syscall.Stat_t
	if err := syscall.Lstat(path, &st); err != nil {
		return 0, err
	}
	return uint64(st.Dev), nil
}
