# go-file-dedup

A fast CLI that finds duplicate files under a directory. Built in Go for
concurrent hashing of large trees.

## Problem

Duplicate files quietly eat disk space — old downloads, copied photo sets,
vendored archives. Finding them naively means hashing every file, which is
slow on big trees. `dedup` avoids most of that work by grouping files by
size first: a file whose size is unique can never have a duplicate, so it is
never hashed.

## Design decisions

1. **Size-bucketing before hashing.** Files are grouped by size during the
   walk; only buckets with 2+ files proceed to hashing. On a typical tree
   this skips the expensive SHA-256 pass for the vast majority of files.
2. **Bounded worker pool for hashing.** A fixed number of goroutines
   (default: CPU count, tunable with `-workers`) digest files over channels.
   The pool is bounded so scanning a directory with a million files doesn't
   spawn a million goroutines or exhaust file descriptors.
3. **Streaming hashes.** `io.Copy` feeds each file through SHA-256 in
   chunks, so multi-gigabyte files are hashed without loading them into
   memory.
4. **Cancellation on error.** The workers share a `context`; the first hash
   failure cancels the remaining work instead of letting it grind on.

## Run

```bash
go build -o dedup ./cmd/dedup

# scan the current directory
./dedup -dir ~/Downloads

# tune parallelism, ignore files under 1 KB, machine-readable output
./dedup -dir /data -workers 8 -min-size 1024 -json
```

Symlinks and directories are ignored; only regular files are considered.

## Tests

```bash
go test ./...
```

Covers: size-bucketing correctness (same-size-but-different-bytes files are
not grouped), duplicate detection, min-size filtering, empty and missing
directories, hash determinism, and the concurrent hashing path.

## Tech stack

Go (standard library only — no external dependencies).
