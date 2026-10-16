package dedup

// Options controls how FindDuplicates scans and hashes files.
type Options struct {
	// Workers is the number of concurrent hashing goroutines.
	// Zero or negative means runtime.NumCPU().
	Workers int
	// MinSize skips files smaller than this many bytes. Files of size 0
	// are skipped unless MinSize is 0.
	MinSize int64
}

// DuplicateGroup is a set of files that are byte-for-byte identical.
type DuplicateGroup struct {
	Hash  string   // SHA-256 hex digest of the file contents
	Size  int64    // Size in bytes of each file
	Files []string // Paths of the identical files, sorted
}

// WastedBytes returns how much disk could be reclaimed by keeping one copy.
func (g DuplicateGroup) WastedBytes() int64 {
	if len(g.Files) < 2 {
		return 0
	}
	return g.Size * int64(len(g.Files)-1)
}

// DefaultOptions returns sane defaults.
func DefaultOptions() Options {
	return Options{Workers: 0, MinSize: 1}
}
