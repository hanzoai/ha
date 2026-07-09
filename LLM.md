# LLM.md — ha

`github.com/hanzoai/ha` — Hanzo's HA coordination primitive. Single-writer election
over a live membership set via Rendezvous (HRW) hashing. No coordinator, lock
service, Postgres, or Redis. Pure Go, stdlib-only (`crypto/sha256`,
`encoding/binary`, `context`), zero external deps.

## What it is / is not
- **Is:** the answer to "who is the single writer for key K right now", identical on
  every replica. `Owner` / `IsOwner` / `Replicas` + the `Membership` seam + `Static`.
- **Is NOT:** storage, replication, or a lock service. It decides WHO writes, never
  HOW state is stored/shipped. That is `hanzoai/vfs` (SQLite + object store), which
  composes this for its writer gate.

## Provenance
Extracted verbatim from `hanzoai/vfs/replica/owner.go` (HIP-0107), which is
storage-agnostic pure Go and had no vfs-internal callers — it was hosted in a SQLite
library only for historical reasons. Promoting it to its own package is the
decomplection: coordination (this) vs replication (vfs) are orthogonal concerns.
The `key` argument generalizes the original `orgID` (any stable ownership unit: org,
shard, or singleton job name).

## Consumers
- `hanzoai/visor` billing (exactly-once hourly metering): `ha.IsOwner("_global", …)`
  gates lease claims so exactly one replica bills per wall-clock hour.
- Adoption target (HIP-0107): iam / commerce / base and any singleton workload.

## Roadmap
- `ha/k8s` (Phase 2): the cluster `Membership` — lists Running+Ready pods by label
  via the in-cluster K8s API. Currently lives as `k8sMembership` inside visor; folding
  it here makes the K8s lister the ONE shared source (no per-service copy).

Update THIS file for notes; do not add scratch summary files.
