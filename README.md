# DistFS

*A work-in-progress personal distributed file system for learning distributed systems by building one I actually use.*

Updated at: May 15, 2026

DistFS is a single-user distributed storage system focused on understanding real distributed systems concepts — consensus, consistency models, replication, locking, and fault tolerance — through implementation, not just theory.

This project exists for two reasons:

1. Re-learn distributed systems deeply enough to reason about them without notes.
2. Build a personal cloud storage system I actually trust with my own data.

The product is the forcing function for the learning.

---

## Current Status

🚧 **Work in progress — early architecture + implementation phase**

Current focus:
- 3-node Raft cluster
- replicated metadata service
- CLI filesystem operations
- fault tolerance and leader election

Planned later:
- FUSE mount
- chunked file storage
- LAN deployment across real machines
- Tailscale access
- encryption support

---

## Goals

DistFS aims to provide:

- Strongly consistent metadata
- Replicated file storage
- Single-user edit-in-place workflows
- Fault tolerance across 3 nodes
- Practical filesystem semantics
- Explicit CAP tradeoffs

This is intentionally:
- **not** multi-user
- **not** offline-first
- **not** a Dropbox clone
- **not** a research project inventing new algorithms

The goal is to use existing ideas (Raft, replication, linearizable metadata) correctly and understand them deeply.

---

## Planned Architecture

```text
          Client (CLI / FUSE)
                   |
                   v
        +-------------------+
        | Raft Metadata     |
        | Cluster (3 nodes) |
        +-------------------+
                   |
                   v
          Replicated Chunks
