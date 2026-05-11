# DistFS — Learning + Product Goals

*Personal distributed file storage system. One-page goals doc. Owned by me, edited by me.*

*Last updated: May 2026*

---

## Why this project exists

Two equally-weighted reasons:

1. **Rego on distributed systems concepts I "got the grade on" but never internalized** — especially consensus, consistency models, and CAP/PACELC tradeoffs. Class projects taught me Paxos mechanics; I want to learn the consistency *language* that explains what guarantees a system actually provides.
2. **Replace iCloud / Drive with something I built**, that I actually use, and trust enough to put my real data on.

If forced to choose, I optimize for *learning that produces a useful artifact* — not "ship fast" and not "academic exercise." The product is the forcing function for the learning.

---

## Learning goals (priority ordered, falsifiable)

**L1. Re-derive consensus from scratch via Raft, not by reading Paxos slides.**
- Success: I can sit at a whiteboard and explain leader election, log replication, and safety properties without notes. I can articulate which parts of Raft are different from Paxos and *why* Ongaro made those choices.
- Falsifier: If I can't explain why a 2-node cluster doesn't actually give fault tolerance — without notes — I haven't learned this.

**L2. Internalize consistency models well enough to argue about them.**
- Success: I can say what consistency model my system provides for each operation (metadata read, metadata write, file read, file write) and defend the choice. I can explain the difference between linearizability, sequential consistency, and eventual consistency in concrete terms, with examples from my own code.
- Falsifier: If someone asks "is your system linearizable?" and I have to look it up, I haven't learned this.

**L3. Make CAP tradeoffs consciously instead of by accident.**
- Success: I have written down, somewhere in my code's design doc, the specific moments my system chooses C over A (or vice versa) and why. When my system blocks during a partition, I can explain why that's the correct behavior given my choices.
- Falsifier: If I can't point to a specific line of code or design decision and say "this is where I chose C over A," it's accidental, not learned.

**L4. Practice the network filesystem concurrency patterns again, in a real system.**
- Success: I implement at least one piece of fine-grained locking (likely on the metadata tree) and can explain why it's correct under concurrent access. I understand the RPC semantics (at-most-once, exactly-once) my system uses and where each applies.
- Falsifier: If I just slap a single global mutex on the metadata service and call it done, I'm not exercising this.

**L5. Get Lamport-clock-ish reasoning back in my hands.**
- Success: I understand and can explain how my Raft log indices function as a logical clock for the metadata, and where pure wall-clock time would be wrong.

**Deprioritized (may show up incidentally, not learning goals):** thread library internals, pager / VM, primary-backup, MapReduce framework, GFS-as-paper-to-reimplement.

---

## Product goals

**P1. Single-user file storage I actually use.** Not multi-tenant. Other people who want this run their own instance.

**P2. FUSE-mounted folder, edit-in-place (eventual). CLI access first.** Tier 1: CLI tool (`distfs put/get/ls`). Tier 2+: Files appear at `~/distfs/` on my laptop, editable via my normal editor with writes going through to the cluster. No sync daemon, no offline mode.

**P2b. Files feel local when editing.** Client caches files it has open; writes go through to the cluster on save. Cache is invalidated when the cluster's version of the file changes.

**P3. Strong consistency on metadata, replicated bulk data.** When I list a directory, I see the truth. When I read a file I just wrote, I see what I wrote. (This is what L2 and L3 force me to actually deliver — the system follows from the learning goals.)

**P4. Multi-machine fault tolerance.** 3 nodes on LAN to start. Killing any 1 node does not lose data and does not block reads. Writes block briefly during leader re-election (acceptable per L3).

**P5. Sub-3-second responsiveness for normal operations.** Listing a directory <1s, opening a small file <2s, starting a stream of a large file <3s.

**P6. Designed to grow to ~5TB over a year, with chunking when files get large.** No erasure coding for v1.

**P7. E2E-compatible architecture; transport + at-rest encryption from day 1.**
- Tailscale handles transport (Wireguard tunnels) once I leave LAN.
- LUKS / FileVault handles at-rest on each node's data drive.
- Server treats file contents as opaque bytes (no server-side content indexing) so E2E remains addable without architectural refactor.

---

## Anti-goals (explicit non-features)

**A1. Not multi-user.** No auth, no ACLs, no quota, no identity in v1. If I want to share with my friend, he runs his own instance.

**A2. Not real-time collaboration.** No Google-Docs-style concurrent editing. One writer at a time per file (enforced by metadata lock).

**A3. Not offline-capable.** If I'm disconnected from the cluster, I can't access files. This is a *feature* — it's what lets me have strong consistency without conflict resolution.

**A4. Not a Drive feature competitor.** No sharing links, no comments, no in-app preview, no mobile app, no thumbnails (initially).

**A5. Not production-grade.** Best-effort reliability with documented failure modes. I keep my real backups elsewhere until I trust this.

**A6. Not a research project.** I am not inventing a new consensus algorithm or a new consistency model. I'm using existing ones (Raft, linearizable metadata) correctly.

**A7. Static cluster membership in v1.** No dynamic add/remove of nodes. Adding a 4th node or moving the cluster requires stopping everything, editing config, restarting. Dynamic membership (joint consensus) is a stretch learning project, not month 1.

---

## Month 1 Definition of Done — three tiers

**Tier 1 (floor — would not be embarrassed):**
3 Raft nodes running as separate processes on my laptop. Replicated key-value store. I can `put`/`get` via a CLI. Killing any 1 node doesn't break reads. Leader election works. *This is the consensus core; if I only get here, project is still a real learning win.*

**Tier 2 (target — "real thing"):**
Tier 1 + 3 nodes on actual separate machines on my LAN + FUSE mount on my laptop. `cp file ~/distfs/` writes the file replicated across all 3 nodes. Killing any 1 node, reads still work, writes resume after election. No chunking, no Tailscale, no UI yet.

**Tier 3 (stretch — likely month 2):**
Tier 2 + Tailscale set up so I can hit the cluster from outside LAN + chunking for files >4MB + at-rest encryption configured.

**I commit to Tier 1 as the floor, aim at Tier 2.** Check progress at end of week 2. If tracking behind, I cut Tier 2 scope, not learning depth.

---

## Decisions made and why (so future me remembers)

- **Raft over Paxos:** chose Raft because I already implemented Paxos for class — the learning value is in *consensus*, and Raft is the modern teaching version that's easier to reason about (Ongaro's whole point). I'll read his thesis as I implement.
- **Single-user over multi-user:** multi-user is ~30% more code with zero distributed-systems learning value. Defer.
- **FUSE edit-in-place over Dropbox-style sync:** sync engines are their own deep project (conflict resolution, change detection). Edit-in-place matches the "consistency over availability" choice and removes a whole problem domain.
- **LAN before Tailscale:** Tailscale is a 10-minute install. Getting consensus + FUSE working on LAN is the actual hard part. Sequence by what's load-bearing.
- **Metadata-only consensus, not all-bytes consensus:** consensus is expensive; pushing every byte of a 4GB video through Raft is the wrong tool. Standard GFS-lineage split. (See L4 — this is also what makes the locking interesting.)
- **No erasure coding in v1:** 3x replication is simple and within my storage budget. Erasure coding is real engineering work (Reed-Solomon, recovery math) that doesn't add new *consensus* or *consistency* learning. Add later if storage cost actually bites.
- **E2E-compatible but not E2E-implemented:** keeps the door open without paying the cost now. The cost of E2E isn't the encryption itself, it's that server-side features (search, dedup, indexing) become impossible. As long as I don't build server-side content indexing, I can add E2E later.

---

## What this doc is for

When I'm tempted to add a feature, I check anti-goals first. When I'm tempted to skip the "boring" learning rep, I check the falsifiers in L1-L5. When I'm 3 weeks in and feel lost in code, I re-read the tier definitions to remember what I'm aiming at.

This doc gets updated when reality changes my mind, not when I drift. Drift goes in a separate "things I want to do later" file, not here.
