// Command dedup finds duplicate files under a directory and reports them.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"runtime"

	"github.com/Subhanmalik138/go-file-dedup/dedup"
)

func main() {
	var (
		dir     = flag.String("dir", ".", "directory to scan for duplicates")
		workers = flag.Int("workers", runtime.NumCPU(), "number of concurrent hashing workers")
		minSize = flag.Int64("min-size", 1, "ignore files smaller than this many bytes")
		asJSON  = flag.Bool("json", false, "print results as JSON")
	)
	flag.Parse()

	groups, err := dedup.FindDuplicates(context.Background(), *dir, dedup.Options{
		Workers: *workers,
		MinSize: *minSize,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "dedup: %v\n", err)
		os.Exit(1)
	}

	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(groups); err != nil {
			fmt.Fprintf(os.Stderr, "dedup: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if len(groups) == 0 {
		fmt.Println("No duplicate files found.")
		return
	}

	for _, g := range groups {
		fmt.Printf("== %d identical files, %d bytes each (%.1f KB reclaimable) ==\n",
			len(g.Files), g.Size, float64(g.WastedBytes())/1024)
		fmt.Printf("   sha256: %s\n", g.Hash)
		for _, f := range g.Files {
			fmt.Printf("   %s\n", f)
		}
		fmt.Println()
	}
	fmt.Printf("Found %d duplicate group(s); %.1f KB could be reclaimed by keeping one copy each.\n",
		len(groups), float64(dedup.TotalWastedBytes(groups))/1024)
}
