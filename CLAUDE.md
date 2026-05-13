# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project context

DistFS is a single-user distributed filesystem being built as a **learning-driven project** — the product (file storage) is a vehicle for internalizing distributed-systems concepts (Raft consensus, consistency models, replication, FUSE). Authoritative scope and motivation live in `docs/goals.md`; the active milestone is described in `docs/slice1.md`. Read both before proposing non-trivial changes.

The project is currently in **Slice 1**: single-node gRPC server + CLI client, no distributed-systems code yet. The goal of this slice is correct streaming I/O, atomic writes, and clean RPC plumbing — Raft, replication, FUSE, and encryption come in later slices.

## Collaboration style

The repo owner is using this project as a learning vehicle and prefers to think through design decisions together rather than receive complete implementations. Default behavior:

- For design questions, surface the tradeoffs and let the user decide. Do **not** present a finished solution unprompted.
- When code is genuinely requested, write the minimum needed to answer the immediate question — not the surrounding refactor.
- Do not add fields, abstractions, or future-proofing without a concrete consumer being implemented in the same change. This principle has come up repeatedly and is the project's most important design discipline.

## Architectural decisions worth knowing

The schema in `proto/v1/distfs.proto` encodes choices that aren't obvious from reading the file alone:

**`FileInfo` is intentionally dual-purpose.** It carries client-supplied metadata as the first message of a `PutRequest` stream *and* server-stored data inside `ListResponse`. Same fields, different semantics by direction:

- In `PutRequest`, `size_bytes` is an *advisory hint* the server uses only for an upfront capacity check (e.g. `statfs`). The stored size is whatever bytes actually arrive — never reject on mismatch.
- In `PutRequest`, `last_modified` is the client's local mtime, stored verbatim and echoed back. **It is not used for ordering, staleness, idempotency, or any validation.** Do not introduce logic that compares it against existing server state.

If a change starts wanting validation gated on these fields, that's a design discussion, not a fix.

**Put flow is client-streaming with a `oneof` first-message payload.**
- Message 1: `FileInfo` — filename + advisory size + client mtime.
- Messages 2..N: `chunk_data` (raw bytes).
- Server writes to a per-request unique temp filename and renames on clean `io.EOF`. Concurrent same-name Puts each get their own temp; last rename wins. Errors mid-stream → delete the temp.

**Filenames may contain `/`.** The server treats them as relative paths under the data directory; the OS creates parent directories automatically. `..` and other path-traversal sequences must be rejected at the server boundary. Directories are **not first-class entities** — they appear implicitly via filenames, and `List` returns a flat list of full relative paths (like `find . -type f`), not a directory-scoped view.

**Ordering and concurrency.** In Slice 1, the only ordering primitive is server-side rename completion order — last-rename-wins. Do not introduce client-supplied versioning or wall-clock comparisons for ordering. The correct ordering primitive arrives with Raft log indices in later slices (see L5 in `goals.md`).

## Build and tooling

Build infrastructure (`go.mod`, `Makefile`) is still being scaffolded. Established choices:

- Go module path: `github.com/kyukibug/distfs`
- Proto package: `distfs.v1`, file at `proto/v1/distfs.proto`
- `option go_package = "github.com/kyukibug/distfs/proto/v1;distfspb"`
- Plain `protoc` toolchain (not Buf)

Proto generation command (from `slice1.md`):

```
protoc --go_out=. --go-grpc_out=. proto/v1/distfs.proto
```

Other choices discussed but not yet wired: stdlib `slog` for logging, `cobra` for the CLI.

## Anti-goals (from `goals.md`)

The following are explicit non-features and should not creep into scope:

- Multi-user, auth, ACLs, quotas
- Real-time collaboration
- Offline-capable client
- Sharing links, comments, mobile clients
- Inventing new consensus algorithms (use existing Raft)
- Dynamic cluster membership (v1)
