// cmd/lbfq/main.go
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"lbfq/internal/scan"
	"lbfq/internal/topn"
	"lbfq/internal/units"
)

func main() {
	root := flag.String("root", "/", "directory to scan")
	topN := flag.Int("n", 50, "show top N largest files")
	minStr := flag.String("min", "0", "only list files >= size (e.g. 100M, 1G)")
	xdev := flag.Bool("xdev", true, "stay on the same filesystem")
	apparent := flag.Bool("apparent", false, "use apparent size instead of on-disk bytes")
	workers := flag.Int("workers", 0, "concurrent stat workers")
	ndjson := flag.Bool("ndjson", false, "print results as newline-delimited JSON(NDJSON) format")
	flag.Parse()

	minSize, err := units.ParseSize(*minStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid -min: %v\n", err)
		os.Exit(2)
	}

	cfg := scan.Config{
		Root:     *root,
		MinSize:  minSize,
		XDev:     *xdev,
		Apparent: *apparent,
		// NOTE: 0 workers means auto-tune based on CPU cores
		Workers: *workers,
		// NOTE:
		// Hard-coded skips for common virtual filesystems.
		// They're usually not interesting for disk usage analysis.
		Skips: []string{"/proc", "/sys", "/run", "/dev"},
	}

	keeper := topn.NewKeeper(*topN)

	for r := range scan.Scan(cfg) {
		keeper.Consider(topn.Item{Size: r.Size, Path: r.Path})
	}

	items := keeper.ItemsDesc()

	// NDJSON output
	if *ndjson {
		enc := json.NewEncoder(os.Stdout)
		// For deterministic key order, encode a struct or map literal;
		type rec struct {
			SizeBytes int64  `json:"size_bytes"`
			SizeHuman string `json:"size_human"`
			Path      string `json:"path"`
		}

		for _, it := range items {
			_ = enc.Encode(rec{
				SizeBytes: it.Size,
				SizeHuman: units.Human(it.Size),
				Path:      it.Path,
			})
		}
		return
	}

	// default text output
	for _, it := range items {
		fmt.Printf("%12s  %s\n", units.Human(it.Size), it.Path)
	}
}
