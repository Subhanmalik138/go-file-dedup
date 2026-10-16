package dedup

import (
	"io/fs"
	"path/filepath"
)

// sizeBuckets walks root and groups regular files by size. Only buckets
// holding two or more candidates are returned, since a file with a unique
// size can never have a duplicate. Symlinks and directories are ignored.
func sizeBuckets(root string, minSize int64) (map[int64][]string, error) {
	buckets := make(map[int64][]string)

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.Type().IsRegular() {
			return nil // skip dirs, symlinks, devices
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.Size() < minSize {
			return nil
		}
		buckets[info.Size()] = append(buckets[info.Size()], path)
		return nil
	})
	if err != nil {
		return nil, err
	}

	for size, files := range buckets {
		if len(files) < 2 {
			delete(buckets, size)
		}
	}
	return buckets, nil
}

// candidates flattens the size buckets into a plain file list.
func candidates(buckets map[int64][]string) []string {
	var out []string
	for _, files := range buckets {
		out = append(out, files...)
	}
	return out
}
