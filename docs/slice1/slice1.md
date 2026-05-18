# Slice 1: Project set-up and single-node put/get

Updated at: May 10, 2026

*Goal: a 1-process server and a CLI client that can store and retrieve files. No replication, no chunking, no Raft, no anything else. Bytes over the wire to disk.*

---

## Why this slice exists

Before any distributed-systems concept enters the codebase, the core functions of the system need to work: gRPC over TCP, file I/O, error propagation, CLI ergonomics, project layout. Slice 1 forces all of that into place while the rest of the system is trivial.

If at the end of slice 1 the `distfs put` and `distfs get` round-trip a file correctly, then every later slice can assume "the wires work" and we can focus on the actual hard parts (consensus, replication, chunking) without simultaneously debugging the RPC plumbing.

---

## What we're building

Two binaries:

- **`distfs-server`** — listens on a TCP port, stores files in a directory on local disk.
- **`distfs`** (the client) — CLI that talks to the server.

Three operations to start:

- `distfs put <localpath> <remotename>` — uploads a file
- `distfs get <remotename> <localpath>` — downloads a file
- `distfs ls` — lists files on the server

Everything runs on the laptop in slice 1. No multi-machine yet.

---

## Architectural shape

A few decisions worth making *now* even though they look like overkill for one server:

1. **Use gRPC with a `.proto` file**, not hand-rolled HTTP. You'll regenerate code as the API evolves; the schema is the interface contract.
2. **Separate `server` package from `proto` package from `client` package.** Even though slice 1 is small, this layout maps directly to later slices where the server becomes multiple services.
3. **The server stores files in a configurable directory**, not a hardcoded path. We'll want to run multiple instances on the same machine later in the project.
4. **Don't read whole files into memory.** Use streaming gRPC for `Put` and `Get`. In slice 1, the files will be small, but we'll have multi-GB files later, and rewriting a streaming layer in a later slice is way worse than getting it right now.
5. **Errors come back as gRPC status codes**, not strings. Use `codes.NotFound`, `codes.AlreadyExists`, `codes.Internal`, etc. Future code will inspect these.

These are the only "build for later" decisions made in Slice 1. Prioritize finishing the goals in Slice 1 before adjusting anything else.

---

## The interface (proto schema)

We design the actual `.proto`. The shape we're aiming for:

```
service DistFS {
  rpc Put(stream PutRequest) returns (PutResponse);
  rpc Get(GetRequest) returns (stream GetResponse);
  rpc List(ListRequest) returns (ListResponse);
}
```

Decisions to make:

- **Streaming chunk size.** What's the size of each `PutRequest`/`GetResponse` payload? (Note: gRPC has a default 4MB message limit. Stay well under it. 64KB-256KB seems reasonable.)
- **How do we frame a file in the stream?** First message has metadata (filename, size), subsequent messages have just bytes? Or every message has the filename and a sequence number? (Note: the first may be simpler.)
- **Atomicity.** What happens if the client disconnects mid-`Put`? Does the server keep the partial file, or discard it? (Note: write to a temp file, rename on success. This is the same pattern as POSIX atomic file writes.)
- **Overwrite semantics.** What if the file already exists when `Put` is called? Reject? Replace? Version? (Note: pick one explicitly; document the choice in our proto comments.)

---

## Repository layout (suggested by Claude)

This can be subject to change.

```
distfs/
├── README.md                  # what the project is
├── WORKLOG.md                 # 3-sentence notes per session
├── docs/
│   ├── goals.md               # the goals doc we wrote
│   └── slice-1.md             # this file
├── proto/
│   └── distfs.proto           # the schema
├── go.mod
├── go.sum
├── cmd/
│   ├── distfs/                # the CLI client binary
│   │   └── main.go
│   └── distfs-server/         # the server binary
│       └── main.go
├── internal/
│   ├── server/                # server implementation
│   ├── client/                # client library (used by CLI)
│   └── storage/               # local disk storage abstraction
└── test/
    └── integration_test.go    # spins up server, runs client commands
```

Note: The `internal/storage` package is doing real work here. Even though slice 1's storage is "write bytes to a file in a directory," giving it its own package means implementing a Raft-replicated storage can plug in without touching client code. **Design the interface, then implement the trivial version.**

---

## Definition of done for slice 1

We're done with slice 1 when all of the following are true:

1. `distfs-server --data-dir /tmp/distfs-data --port 9000` starts a server.
2. `distfs --addr localhost:9000 put ~/photo.jpg vacation/photo.jpg` uploads a file and exits with code 0 on success.
3. `distfs --addr localhost:9000 get vacation/photo.jpg ~/recovered.jpg` downloads it and the byte content matches the original (`diff` shows no difference, or `sha256sum` matches).
4. `distfs --addr localhost:9000 ls` shows `vacation/photo.jpg`.
5. Errors are sensible: getting a nonexistent file returns a clear "not found" error, not a panic.
6. An integration test exists that exercises put → ls → get against a real server (started by the test), and passes.
7. A file > 100MB round-trips correctly without OOMing the server or client (proves streaming actually works).
8. The server can be killed and restarted; previously-uploaded files are still retrievable (proves files actually hit disk, not just memory).

None of these goals prioritize performance, this will be part of a later milestone.

---

## Things explicitly NOT in slice 1

To stop ourselves from scope creep, here's what doesn't belong:

- Authentication, TLS, encryption
- Multiple servers, replication, consensus
- Chunking
- Concurrency on the server beyond what gRPC provides for free (just let gRPC's worker model handle it)
- A nice TUI / progress bars
- Anything in `~/distfs/` mounted as a folder
- Quota management, disk usage tracking
- Garbage collection of orphaned files
- Caching on the client side

---

## Open questions

These are the design decisions in slice 1 we have deliberately left open-ended:

1. **gRPC server lifecycle:** how should the server handle graceful shutdown? Signal handling, draining in-flight RPCs?
2. **Filename validation:** should we allow `..` in remote paths? Special characters? Null bytes? (Note: validate aggressively. Reject anything weird with `codes.InvalidArgument`.)
3. **List output format:** just names? Names + sizes + mtimes? (Note: maybe include sizes and timestamps for FUSE implementation.)
4. **Logging:** how does the server log? stdlib `log`, `slog`, zap? (Note: `slog` is in the stdlib is fine for now.)
5. **Configuration:** flags? env vars? a config file? (Note: flags for now. Config file when we have >5 settings.)

---

## Suggested working order - Subject to change

1. Write the `.proto` file. Get `protoc` generating Go code.
2. Write `internal/storage` interface and its single-disk implementation. Unit tests for storage.
3. Write `internal/server` that wires storage to the gRPC service. No client yet.
4. Manually test with `grpcurl` to verify server works in isolation.
5. Write `internal/client` library.
6. Write `cmd/distfs` CLI on top of the client library.
7. Write integration test.
8. Make the 100MB streaming test pass (Note: this can help catch buffer-size assumptions).
9. Commit, update WORKLOG.md, move to slice 2.

---

## What slice 2 might look like (preview, don't build yet)

So we can see where this is heading and design slice 1's interfaces to fit:

In slice 2, `distfs-server` splits into `distfs-metadata` and `distfs-chunkstore`. The CLI client first calls metadata to ask "where do I put this file?", gets back a placement decision ("write to chunkstore at localhost:9001"), then streams bytes to the chunkstore. The metadata service tracks file names → chunk locations.

This means in slice 1, we should keep our `internal/storage` interface narrow enough that it could be replaced by "ask another service where to put this." Don't bake file naming logic into the storage layer. Don't have the storage layer talk to gRPC. Keep concerns separated.
