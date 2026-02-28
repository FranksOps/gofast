Phase 1: The Abstraction Layer

- [x] Define the Provider interface:
      ```go
      type Provider interface {
          Stat(path string) (FileInfo, error)
          List(path string) ([]FileInfo, error)
          OpenRead(path string) (io.ReadCloser, error)
          OpenWrite(path string, metadata FileInfo) (io.WriteCloser, error)
      }
      ```
- [x] Implement LocalProvider (POSIX) as the baseline.

Phase 2: The Concurrent Engine

- [x] Implement a Work Channel that accepts TransferJob structs.
- [x] Build the Dynamic Worker Pool: Workers can be added or decommissioned gracefully without stopping the entire migration.
- [x] Buffer Management: Use sync.Pool for reusable byte buffers to keep Garbage Collection (GC) overhead low during multi-terabyte transfers.

Phase 3: Resilience & Resumability

- [ ] Checkpointing: Every X megabytes or per-file, update the local State Store.
- [ ] The "Deep Walk" Logic: Instead of recursive function calls (which risk stack overflow on deep paths), use a stack-based iterative walker.

Phase 4: Advanced Storage Features

- [ ] Metadata Mapping: Logic to handle chown/chmod across different filesystems (e.g., translating "Root" on Source A to a specific "UID" on Destination B).
- [ ] S3/Object Support: Implement the Provider interface for S3-compatible storage.

Phase 5: User Interface (TUI)

- [ ] Develop a bubbletea (or similar) based Terminal UI to show:
  - [ ] Active streams and their current file/speed.
  - [ ] Total ETA based on a weighted average of recent throughput.
  - [ ] Visual "hot-key" menu to scale streams up or down on the fly.
