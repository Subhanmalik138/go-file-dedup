package dedup

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"
)

// HashFile streams a file through SHA-256 and returns the hex digest.
// The file is never loaded fully into memory.
func HashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("hash %s: %w", path, err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// hashResult carries one file's digest (or its error) back to the collector.
type hashResult struct {
	path string
	hash string
	err  error
}

// HashFiles digests every path with a bounded pool of workers. On the first
// hash error the remaining work is cancelled via the context.
func HashFiles(ctx context.Context, paths []string, workers int) (map[string][]string, error) {
	if workers <= 0 {
		workers = runtime.NumCPU()
	}

	jobs := make(chan string)
	results := make(chan hashResult)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range jobs {
				hash, err := HashFile(path)
				select {
				case results <- hashResult{path: path, hash: hash, err: err}:
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	go func() {
		defer close(jobs)
		for _, p := range paths {
			select {
			case jobs <- p:
			case <-ctx.Done():
				return
			}
		}
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	byHash := make(map[string][]string)
	for r := range results {
		if r.err != nil {
			cancel()
			return nil, r.err
		}
		byHash[r.hash] = append(byHash[r.hash], r.path)
	}
	return byHash, nil
}
