package dedup

import (
	"context"
	"runtime"
	"sort"
)

// FindDuplicates scans root and returns groups of byte-identical files.
// It works in two phases: files are first bucketed by size (a file with a
// unique size cannot be a duplicate and is never hashed), then the
// remaining candidates are hashed concurrently with a worker pool.
func FindDuplicates(ctx context.Context, root string, opts Options) ([]DuplicateGroup, error) {
	if opts.Workers <= 0 {
		opts.Workers = runtime.NumCPU()
	}

	buckets, err := sizeBuckets(root, opts.MinSize)
	if err != nil {
		return nil, err
	}

	var groups []DuplicateGroup
	for size, files := range buckets {
		byHash, err := HashFiles(ctx, files, opts.Workers)
		if err != nil {
			return nil, err
		}
		for hash, paths := range byHash {
			if len(paths) < 2 {
				continue
			}
			sort.Strings(paths)
			groups = append(groups, DuplicateGroup{Hash: hash, Size: size, Files: paths})
		}
	}

	sort.Slice(groups, func(i, j int) bool {
		if groups[i].Size != groups[j].Size {
			return groups[i].Size > groups[j].Size // biggest waste first
		}
		return len(groups[i].Files) > len(groups[j].Files)
	})
	return groups, nil
}

// TotalWastedBytes sums the reclaimable bytes across all groups.
func TotalWastedBytes(groups []DuplicateGroup) int64 {
	var total int64
	for _, g := range groups {
		total += g.WastedBytes()
	}
	return total
}
