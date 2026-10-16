package dedup

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fixture builds a temp tree with known duplicates:
//   a.txt, sub/b.txt   identical  -> duplicate group of 2
//   c1.bin, c2.bin, c3.bin identical -> duplicate group of 3
//   uniq.txt           same size as a.txt but different bytes
//   lone.txt           unique size, never hashed
func fixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	write := func(name, content string) {
		t.Helper()
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	write("a.txt", "hello world")
	write("sub/b.txt", "hello world")
	write("c1.bin", "0123456789abcdef")
	write("c2.bin", "0123456789abcdef")
	write("c3.bin", "0123456789abcdef")
	write("uniq.txt", "HELLO WORLD") // same size as a.txt, different bytes
	write("lone.txt", "i am the only one of my size 1234567890")
	return dir
}

func TestFindDuplicates(t *testing.T) {
	dir := fixture(t)

	groups, err := FindDuplicates(context.Background(), dir, DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}

	if len(groups) != 2 {
		t.Fatalf("expected 2 duplicate groups, got %d", len(groups))
	}

	// Groups are sorted by size desc: the 16-byte bin group comes first.
	if len(groups[0].Files) != 3 || groups[0].Size != 16 {
		t.Errorf("first group should be 3 files of 16 bytes, got %+v", groups[0])
	}
	if len(groups[1].Files) != 2 || groups[1].Size != 11 {
		t.Errorf("second group should be 2 files of 11 bytes, got %+v", groups[1])
	}

	// uniq.txt must NOT be grouped with a.txt despite equal size.
	for _, f := range groups[1].Files {
		if strings.HasSuffix(f, "uniq.txt") {
			t.Errorf("uniq.txt (same size, different bytes) was wrongly grouped: %v", groups[1].Files)
		}
	}
}

func TestWastedBytes(t *testing.T) {
	g := DuplicateGroup{Size: 16, Files: []string{"a", "b", "c"}}
	if got := g.WastedBytes(); got != 32 {
		t.Errorf("expected 32 wasted bytes, got %d", got)
	}
	if got := TotalWastedBytes([]DuplicateGroup{g}); got != 32 {
		t.Errorf("expected total 32, got %d", got)
	}
}

func TestMinSizeSkipsSmallFiles(t *testing.T) {
	dir := fixture(t)

	groups, err := FindDuplicates(context.Background(), dir, Options{Workers: 2, MinSize: 100})
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 0 {
		t.Errorf("expected no groups with MinSize=100, got %d", len(groups))
	}
}

func TestEmptyDirectory(t *testing.T) {
	groups, err := FindDuplicates(context.Background(), t.TempDir(), DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 0 {
		t.Errorf("expected no groups in empty dir, got %d", len(groups))
	}
}

func TestNonexistentRoot(t *testing.T) {
	_, err := FindDuplicates(context.Background(), filepath.Join(t.TempDir(), "nope"), DefaultOptions())
	if err == nil {
		t.Error("expected an error for a nonexistent root, got nil")
	}
}

func TestHashFileDeterministic(t *testing.T) {
	dir := fixture(t)
	h1, err := HashFile(filepath.Join(dir, "a.txt"))
	if err != nil {
		t.Fatal(err)
	}
	h2, err := HashFile(filepath.Join(dir, "a.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if h1 != h2 {
		t.Errorf("hash not deterministic: %s vs %s", h1, h2)
	}
}

func TestHashFilesConcurrency(t *testing.T) {
	dir := fixture(t)
	paths := []string{
		filepath.Join(dir, "a.txt"),
		filepath.Join(dir, "sub", "b.txt"),
		filepath.Join(dir, "c1.bin"),
	}
	byHash, err := HashFiles(context.Background(), paths, 4)
	if err != nil {
		t.Fatal(err)
	}
	if len(byHash) != 2 {
		t.Fatalf("expected 2 distinct hashes, got %d", len(byHash))
	}
	for hash, files := range byHash {
		if len(hash) != 64 { // sha256 hex
			t.Errorf("hash %q is not 64 hex chars", hash)
		}
		_ = files
	}
}
