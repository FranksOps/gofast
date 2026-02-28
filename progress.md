## [Sat Feb 28 15:02:00 EST 2026] Phase 2 - Buffer Management

- Implemented `BufferPool` in `engine/buffer.go` using `sync.Pool`.
- The pool supports configurable buffer sizes with a default of 1MB, ensuring fast I/O throughput out of the box.
- Added corresponding unit tests in `engine/buffer_test.go` to ensure correctness of allocation and recycling.
- *Trade-off / Decision*: Chose a default 1MB buffer size. Large fast transfers benefit from slightly larger buffers than `io.Copy`'s default (32KB). Designed the `Get()` method to return a pointer (`*[]byte`) to avoid copying slice headers unnecessarily during pool retrieval and usage, which helps slightly reduce memory allocations.

## [Date] Phase 3 - Resilience & Resumability

- Implemented internal job state storage using `bbolt` in a new `store` package (`store.go`).
- Added `store_test.go` with full test coverage for the simple key-value state store.
- Added a `JobTracker` and `TrackedWriter` to `engine/tracker.go` to support checkpointing based on bytes written or time elapsed (configurable). 
- Updated `engine.TransferJob` to include an `ID` to map engine state back to the store.
- *Decisions Made*: Chose `go.etcd.io/bbolt` over SQLite since bbolt is pure Go and doesn't require CGO, making the final binary easier to cross-compile for admins. The `TrackedWriter` implements `io.Writer` gracefully around the inner writer (presumably from the Provider), intercepting the byte count, reducing checkpoint DB writes over standard `Write` operations via configurable limits `BytesInterval` (default 10MB) and `TimeInterval` (default 5s).
