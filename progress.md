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

## [Sat Feb 28 15:19:39 EST 2026] Phase 3 - The Deep Walk Logic

- Implemented `Walker` in `engine/walker.go` to traverse directory tree without recursion.
- Used a stack slice to maintain state iteratively, avoiding function call overhead and stack limits for deep directories.
- Tested `Walker` thoroughly with `mockProvider` in `walker_test.go` to ensure all files in deep directories correctly yield a `TransferJob`.
- *Trade-off / Decision*: Using a depth-first traversal with a slice as a stack. This minimizes immediate memory usage vs breadth-first queue, but handles arbitrarily deep directory structures easily. Generates and pushes `TransferJob`s directly to the `JobChannel`. Added cancelation checking between directory reads and job pushes to honor `context.Context`.

## [Sat Feb 28 15:20:28 EST 2026] Phase 4 - Metadata Mapping

- Implemented `MetadataMapper` in `provider/metadata.go` to support UID/GID mapping and translation.
- Added `UnixFileInfo` interface extending `FileInfo` with UID, GID, and Mode to store POSIX metadata.
- Updated `LocalProvider` to extract `syscall.Stat_t` internal metadata when running `Stat()` and `List()`.
- Added logic in `LocalProvider`'s `OpenWrite` and `Close` to apply mapped metadata (chown/chmod) when writing locally.
- *Trade-off / Decision*: Application of metadata currently ignores errors (e.g., typically a non-root user cannot freely `chown` a file they're creating to arbitrary UIDs). Decided to fail silently on permission boundaries during `chown/chmod` metadata application rather than failing the entire transfer, which is standard behavior natively (e.g. standard user cp).
## [Sat Feb 28 15:37:24 EST 2026] Phase 4 - Advanced Storage Features (S3/Object Support)

- Implemented `S3Provider` in `provider/s3.go` supporting `Stat`, `List`, `OpenRead`, and `OpenWrite`.
- Integrated `aws-sdk-go-v2` and features like the `manager.Uploader` for multipart streaming handles.
- *Decisions Made*:
  - Represented directories in S3 logic either implicitly (via CommonPrefixes and `/` suffix keys) or explicitly (via 0-byte objects).
  - Implemented streaming writes for S3 by wiring up an `io.Pipe`, allowing the engine to write continuously while the AWS SDK handles background concurrent multipart uploads seamlessly. 
  - `Stat` does an exact match checked via `HeadObject` before falling back to `ListObjectsV2` for virtual path-directory discovery.
  - Used `github.com/aws/aws-sdk-go-v2/feature/s3/manager` (v1 upload manager) for uploading instead of the v2 transfer manager to avoid dependencies and stability issues since `manager` has long-standing stable support for Go readers.
  - Mapped directories locally via `dummyWriter` for `OpenWrite` when metadata signals a directory placeholder.
