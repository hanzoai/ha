# ha

Hanzo's high-availability coordination primitive: **single-writer election over a
live membership set** — no coordinator, no lock service, no Postgres, no Redis.

It answers one question, identically on every replica:

> which replica is the single writer for key `K` right now?

via Rendezvous (Highest-Random-Weight) hashing. `N` stateless replicas agree on one
owner per key and fail over deterministically, with no recomputation and nothing to
coordinate.

## Why it is its own package

`ha` decides **who writes**. It never touches **how** state is stored or replicated
— that is [`hanzoai/vfs`](https://github.com/hanzoai/vfs) (per-org SQLite + object
store) or any other backing. Keeping election separate means a cron tick, a queue
consumer, or any singleton can be made exactly-once **without importing a storage
library** to get it. One primitive, one home, every consumer.

```
ha            who is the single writer          ← this package (pure Go, zero deps)
vfs           how the SQLite state replicates    → composes ha for its writer gate
your service  what the single writer does        → cron, billing sweep, queue drain
```

## Use

```go
import "github.com/hanzoai/ha"

// The per-write hot path: only the elected owner of key proceeds.
if ha.IsOwner(orgID, self, members) {
    // ... this replica is the single writer for orgID; do the write.
}
```

`members` comes from a `Membership` — the injectable seam:

```go
// Single process / local dev / tests: itself is the sole writer.
m := ha.Static(podName)

// Production: a cluster source (e.g. hanzoai/ha/k8s) lists live, ready peers.
self := m.Self()
members, err := m.Members(ctx)
if err != nil || len(members) == 0 {
    // FAIL CLOSED — unknown or empty membership has no safe owner; skip.
    return
}
if ha.IsOwner(orgID, self, members) { /* write */ }
```

## API

| Symbol | Purpose |
|--------|---------|
| `Member{ID, Addr}` | one replica; `ID` is the stable HRW identity, `Addr` for write-forwarding |
| `Owner(key, members) (Member, bool)` | the elected single writer for `key`; `ok=false` on empty set (fail-closed) |
| `IsOwner(key, self, members) bool` | the per-write gate every replica runs |
| `Replicas(key, members, n) []Member` | owner first, then ordered failover successors (pre-warm order) |
| `Membership` | seam: `Self()` + `Members(ctx)` — the live set election elects over |
| `Static(id)` | single-process `Membership` (one member, itself, never an error) |

## Guarantees

- **Deterministic & order-independent** — same `(key, members)` ⇒ same owner on
  every replica, regardless of set order (ties break on `ID`).
- **Fail-closed** — empty membership yields no owner; a consumer that cannot read
  its peers must skip, never assume ownership.
- **Even spread** — distinct keys distribute across replicas (HRW), not all pinned
  to one, so ownership (and its write load) balances.
- **Cheap** — a per-write check is a handful of SHA-256s; see `BenchmarkOwnerElection`.

Apache-2.0.
