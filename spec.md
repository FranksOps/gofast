Project Identity

    Project Name: Gofast

    Binary Name: gfast

    Mission: A high-performance, concurrent, and resumable data migration engine for storage and backup administrators.

Technical Specifications

    Architecture: Provider-Agnostic Engine. The core logic handles the queue and workers; "Providers" handle the specific storage protocols.

    Concurrency Model: * The Dispatcher: A single-threaded, low-memory directory walker.

        The Worker Pool: A dynamic set of goroutines performing the io.CopyBuffer operations.

    Integrity: Mandatory Streaming Checksums. Hashing occurs during the read/write stream to avoid a second I/O pass.

    State Store: Embedded BoltDB or SQLite (WAL mode) to track file status (Pending, In-Progress, Completed, Failed).

    Performance Tuning: Real-time adjustment of WorkerCount via stdin or Unix signals (e.g., USR1 to increase, USR2 to decrease).