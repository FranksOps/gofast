Gofast (gfast)
High-Concurrency Data Orchestrator & Migrator

Gofast is a high-performance, Go-native alternative to rsync and fpsync. Built for storage and backup administrators, it handles massive, deeply nested datasets across heterogeneous storage environments (NFS, SMB/CIFS, Block, and Object) with a focus on maximum throughput and stateful resilience.
Core Philosophy

Modern storage migration shouldn't be limited by single-threaded legacy tools. Gofast treats data movement as a scalable pipeline, allowing you to saturate your network pipe while maintaining bit-perfect data integrity.
Key Features

    Dynamic Stream Scaling: Adjust the number of concurrent transfer streams (goroutines) in real-time without restarting the process.

    Protocol Agnostic: Move data seamlessly between different storage technologies using a pluggable Provider architecture.

    Stateful Resumability: Uses a local metadata store to track progress. If a transfer is interrupted, it resumes exactly where it left off—no redundant scanning.

    Deep-Tree Optimization: A stack-based iterative walker designed to handle directory structures hundreds of levels deep without memory exhaustion.

    Streaming Integrity: Integrated checksumming (CRC64/XXHash) performed during the I/O stream to ensure data validty without a secondary read pass.

    Metadata Retention: Optional preservation of POSIX permissions, ownership (UID/GID), and timestamps.

Use Cases

    Data Center Migrations: Moving petabytes of data from legacy NAS to new high-performance flash arrays.

    Cloud On-ramping: Synchronizing local file-based storage to S3-compatible object storage.

    Disaster Recovery: Rapidly restoring deep directory structures over high-latency network links.

    Backup Consolidation: Aggregating disparate storage mounts into a centralized immutable backup repository.

Technical Stack

    Language: Golang (1.21+)

    Concurrency: Worker-pool pattern via Goroutines/Channels

    State Engine: Embedded BoltDB / SQLite (WAL mode)

    Interface: Pluggable Provider system for NFS, SMB, S3, and Local FS.

Quick Start (Conceptual)
Bash

# Basic migration with 32 concurrent streams
gfast sync /mnt/old_nfs /mnt/new_flash --streams 32

# Increase streams to 64 on the fly (via SIGUSR1)
kill -USR1 $(pgrep gfast)

# Resume a previously interrupted transfer
gfast resume --job-id <uuid>