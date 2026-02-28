# Gofast (gfast) - High-Concurrency Data Orchestrator & Migrator

Gofast is a high-performance, Go-native alternative to rsync and fpsync. Built for storage and backup administrators, it handles massive, deeply nested datasets across heterogeneous storage environments (NFS, SMB/CIFS, Block, and Object) with a focus on maximum throughput and stateful resilience.

## Core Philosophy

Modern storage migration shouldn't be limited by single-threaded legacy tools. Gofast treats data movement as a scalable pipeline, allowing you to saturate your network pipe while maintaining bit-perfect data integrity.

## Key Features

- **Dynamic Stream Scaling**: Adjust the number of concurrent transfer streams (goroutines) in real-time without restarting the process.
- **Protocol Agnostic**: Move data seamlessly between different storage technologies using a pluggable Provider architecture.
- **Stateful Resumability**: Uses a local metadata store to track progress. If a transfer is interrupted, it resumes exactly where it left off—no redundant scanning.
- **Deep-Tree Optimization**: A stack-based iterative walker designed to handle directory structures hundreds of levels deep without memory exhaustion.
- **Streaming Integrity**: Integrated checksumming (CRC64) performed during the I/O stream to ensure data validity without a secondary read pass.
- **Metadata Retention**: Optional preservation of POSIX permissions, ownership (UID/GID), and timestamps.
- **Real-time TUI**: Terminal UI showing active streams, throughput, ETA, and worker scaling controls.

## Installation

```bash
go install github.com/franksops/gofast/cmd/gfast@latest
```

## Quick Start

```bash
# Basic migration with 32 concurrent streams
gfast -source /data/old -dest /data/new -streams 32

# Cloud migration with 64 streams and checksum verification
gfast -source /data/local -dest s3://bucket/prefix -streams 64 -checksum

# Resume a previously interrupted transfer
gfast -source /data/old -dest /data/new -state-dir ./gofast-state
```

## Command Line Options

```
-source string
    Source path (local or s3://bucket/prefix)
-dest string
    Destination path (local or s3://bucket/prefix)
-streams int
    Number of concurrent transfer streams (default: 32)
-buffer-size int
    Buffer size in bytes for each stream (default: 1048576)
-state-dir string
    Directory to store state/checkpoint files (default: "./.gofast-state")
-no-metadata
    Disable metadata preservation (UID/GID/mode)
-checksum
    Enable streaming checksum verification (CRC64)
-tui
    Enable TUI (disable for headless operation)
```

## Examples

### Local to Local Migration
```bash
# Migrate /mnt/old to /mnt/new with 64 concurrent streams
gfast -source /mnt/old -dest /mnt/new -streams 64
```

### Local to S3 Migration
```bash
# Upload local data to S3 bucket
gfast -source /data/local -dest s3://mybucket/backup -streams 32
```

### Adjust Streams on the Fly
```bash
# While running, send SIGUSR1 to increase workers, SIGUSR2 to decrease
kill -USR1 $(pgrep gfast)  # Increase workers
kill -USR2 $(pgrep gfast)  # Decrease workers
```

## Architecture

### Provider Abstraction
Gofast uses a Provider interface that abstracts storage backends:
- **LocalProvider**: POSIX-compliant local filesystems
- **S3Provider**: Amazon S3 and S3-compatible storage

### Concurrency Model
- **Dispatcher**: Single-threaded, low-memory directory walker
- **Worker Pool**: Dynamic set of goroutines performing io.CopyBuffer operations
- **Buffer Pool**: Reusable byte buffers via sync.Pool to minimize GC overhead

### State Management
- **Embedded BoltDB**: Tracks file status (Pending, In-Progress, Completed, Failed)
- **Checkpointing**: Periodic state saves (configurable by bytes or time interval)
- **Resumability**: Interrupted transfers resume from last checkpoint

## Use Cases

- **Data Center Migrations**: Moving petabytes of data from legacy NAS to new high-performance flash arrays.
- **Cloud On-ramping**: Synchronizing local file-based storage to S3-compatible object storage.
- **Disaster Recovery**: Rapidly restoring deep directory structures over high-latency network links.
- **Backup Consolidation**: Aggregating disparate storage mounts into a centralized immutable backup repository.

## Technical Specifications

| Feature | Value |
|---------|-------|
| Language | Go 1.25+ |
| Concurrency | Goroutines/Channels |
| State Engine | BoltDB (embedded) |
| Checksum | CRC64 (streaming) |
| Buffer Size | 1MB default (configurable) |
| Default Streams | 32 (configurable) |

## Development

```bash
# Build
go build ./...

# Run tests
go test ./...

# Run with TUI
go run cmd/gfast/main.go -source /tmp/src -dest /tmp/dst -streams 16
```

## License

MIT License - see LICENSE file for details.
