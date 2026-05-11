# Slice 1: Single-node put/get

*Goal: a 1-process server and a CLI client that can store and retrieve files. No replication, no chunking, no Raft, no anything else. Bytes over the wire to disk.*

*Estimated effort: 1-2 sessions, 4-6 hours total.*

---

## Why this slice exists

Before any distributed-systems concept enters the codebase, the boring stuff needs to work: gRPC over TCP, file I/O, error propagation, CLI ergonomics, project layout. Slice 1 forces all of that into place while the rest of the system is trivial.

If at the end of slice 1 your `distfs put` and `distfs get` round-trip a file correctly, then every later slice can assume "the wires work" and you can focus on the actual hard parts (consensus, replication, chunking) without simultaneously debugging your RPC plumbing.

---

## What you're building

Two binaries:

- **`distfs-server`** — listens on a TCP port, stores files in a directory on local disk.
- **`distfs`** (the client) — CLI that talks to the server.

Three operations to start:

- `distfs put <localpath> <remotename>` — uploads a file
- `distfs get <remotename> <localpath>` — downloads a file
- `distfs ls` — lists files on the server

Everything runs on your laptop in slice 1. No multi-machine yet.

---

## Architectural shape (don't break these even though they feel premature)

A few decisions worth making *now* even though they look like overkill for one server:

1. **Use gRPC with a `.proto` file**, not hand-rolled HTTP. You'll regenerate code as the API evolves; the schema is your interface contract.
2. **Separate `server` package from `proto` package from `client` package.** Even though slice 1 is small, this layout maps directly to later slices where the server becomes multiple services.
3. **The server stores files in a configurable directory**, not a hardcoded path. You'll want to run multiple instances on the same machine in slice 3+.
4. **Don't read whole files into memory.** Use streaming gRPC for `Put` and `Get`. Slice 1 files will be small, but you'll have multi-GB files later, and rewriting the streaming layer in slice 5 is way worse than getting it right now.
5. **Errors come back as gRPC status codes**, not strings. Use `codes.NotFound`, `codes.AlreadyExists`, `codes.Internal`, etc. Future code will inspect these.

These are the only "build for later" decisions I'm asking you to make in slice 1. Everything else, optimize for getting it working.

---

## The interface (proto schema)

You design the actual `.proto`. The shape you're aiming for:

```
service DistFS {
  rpc Put(stream PutRequest) returns (PutResponse);
  rpc Get(GetRequest) returns (stream GetResponse);
  rpc List(ListRequest) returns (ListResponse);
}
```

Decisions to make yourself:

- **Streaming chunk size.** What's the size of each `PutRequest`/`GetResponse` payload? (Hint: gRPC has a default 4MB message limit. Stay well under it. 64KB-256KB is reasonable.)
- **How do you frame a file in the stream?** First message has metadata (filename, size), subsequent messages have just bytes? Or every message has the filename and a sequence number? (Hint: the first is simpler.)
- **Atomicity.** What happens if the client disconnects mid-`Put`? Does the server keep the partial file, or discard it? (Hint: write to a temp file, rename on success. This is the same pattern as POSIX atomic file writes.)
- **Overwrite semantics.** What if the file already exists when `Put` is called? Reject? Replace? Version? (Hint: pick one explicitly; document the choice in your proto comments.)

---

## Repository layout (suggested, not mandatory)

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

The `internal/storage` package is doing real work here. Even though slice 1's storage is "write bytes to a file in a directory," giving it its own package means slice 4's "Raft-replicated storage" can plug in without touching client code. **Design the interface, then implement the trivial version.**

---

## Definition of done for slice 1

You're done with slice 1 when all of the following are true:

1. `distfs-server --data-dir /tmp/distfs-data --port 9000` starts a server.
2. `distfs --addr localhost:9000 put ~/photo.jpg vacation/photo.jpg` uploads a file and exits with code 0 on success.
3. `distfs --addr localhost:9000 get vacation/photo.jpg ~/recovered.jpg` downloads it and the byte content matches the original (`diff` shows no difference, or `sha256sum` matches).
4. `distfs --addr localhost:9000 ls` shows `vacation/photo.jpg`.
5. Errors are sensible: getting a nonexistent file returns a clear "not found" error, not a panic.
6. An integration test exists that exercises put → ls → get against a real server (started by the test), and passes.
7. A file > 100MB round-trips correctly without OOMing the server or client (proves streaming actually works).
8. The server can be killed and restarted; previously-uploaded files are still retrievable (proves files actually hit disk, not just memory).

Note that none of these mention performance. Don't optimize. As long as a 100MB file works, you're done.

---

## Things explicitly NOT in slice 1

To stop yourself from scope creep, here's what doesn't belong:

- Authentication, TLS, encryption (slice 7+)
- Multiple servers, replication, consensus (slices 3-4)
- Chunking (slice 5)
- Concurrency on the server beyond what gRPC gives you for free (just let gRPC's worker model handle it)
- A nice TUI / progress bars
- Anything in `~/distfs/` mounted as a folder (slice 6, FUSE)
- Quota management, disk usage tracking
- Garbage collection of orphaned files
- Caching on the client side

If you find yourself reaching for one of these, write it down somewhere ("things I want to do later") and keep moving.

---

## Open questions you'll have to answer yourself

These are the design decisions in slice 1 that I'm deliberately not making for you:

1. **gRPC server lifecycle:** how do you handle graceful shutdown? Signal handling, draining in-flight RPCs?
2. **Filename validation:** do you allow `..` in remote paths? Special characters? Null bytes? (Hint: validate aggressively. Reject anything weird with `codes.InvalidArgument`.)
3. **List output format:** just names? Names + sizes + mtimes? (Hint: include sizes and timestamps. You'll want them in slice 6 for FUSE.)
4. **Logging:** how does the server log? stdlib `log`, `slog`, zap? (Hint: `slog` is in the stdlib now and is fine.)
5. **Configuration:** flags? env vars? a config file? (Hint: flags for now. Config file when you have >5 settings.)

---

## Suggested working order

Engineering practice: build the smallest end-to-end thing first, then expand.

1. Write the `.proto` file. Get `protoc` generating Go code.
2. Write `internal/storage` interface and its single-disk implementation. Unit tests for storage.
3. Write `internal/server` that wires storage to the gRPC service. No client yet.
4. Manually test with `grpcurl` to verify server works in isolation.
5. Write `internal/client` library.
6. Write `cmd/distfs` CLI on top of the client library.
7. Write integration test.
8. Make the 100MB streaming test pass (this often catches buffer-size assumptions).
9. Commit, update WORKLOG.md, move to slice 2.

---

## What slice 2 will look like (preview, don't build yet)

So you can see where this is heading and design slice 1's interfaces to fit:

In slice 2, `distfs-server` splits into `distfs-metadata` and `distfs-chunkstore`. The CLI client first calls metadata to ask "where do I put this file?", gets back a placement decision ("write to chunkstore at localhost:9001"), then streams bytes to the chunkstore. The metadata service tracks file names → chunk locations.

This means in slice 1, you should keep your `internal/storage` interface narrow enough that it could be replaced by "ask another service where to put this." Don't bake file naming logic into the storage layer. Don't have the storage layer talk to gRPC. Keep concerns separated.
