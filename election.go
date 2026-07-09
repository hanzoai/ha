// Copyright 2025 Hanzo AI Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package ha

// Single-writer election WITHOUT a coordinator. Answers one question, identically
// on every replica: "which replica is the single writer for key K?" via Rendezvous
// (Highest-Random-Weight) hashing over the live membership set. No election
// protocol, no lock service, no service discovery, no Postgres, no Redis.
//
// The key is any stable string that names a unit of ownership — an org id (the
// owning replica writes ALL of that org's databases, so the org's data has locality
// and moves as one unit on failover), a shard id, or a singleton job name. Reads are
// served from any replica; only writes are gated. A unit that outgrows one writer is
// a future sub-shard concern (hash key+shard); one-key-one-owner is the correct,
// simple default and the natural isolation boundary.

import (
	"crypto/sha256"
	"encoding/binary"
)

// Member is one replica in the live membership set. ID must be stable for the
// life of the replica (e.g. its pod name / a persistent node id): HRW weights
// derive from it. Addr is the reachable address for write-forwarding (host:port).
type Member struct {
	ID   string
	Addr string
}

// Owner returns the replica that owns the writer for key, or ok=false when members
// is empty (fail-closed — never a wrong writer). Deterministic: the same
// (key, members) yields the same Owner on every replica, independent of order.
func Owner(key string, members []Member) (Member, bool) {
	if len(members) == 0 {
		return Member{}, false
	}
	best := members[0]
	bestScore := weight(key, best.ID)
	for _, m := range members[1:] {
		s := weight(key, m.ID)
		// Tie-break on ID so the result is order-independent.
		if s > bestScore || (s == bestScore && m.ID < best.ID) {
			best, bestScore = m, s
		}
	}
	return best, true
}

// IsOwner reports whether selfID owns the writer for key under members — the
// per-write hot-path check every replica runs.
func IsOwner(key, selfID string, members []Member) bool {
	o, ok := Owner(key, members)
	return ok && o.ID == selfID
}

// Replicas returns key's owner first, then ordered failover successors. On owner
// loss the next replica becomes owner with no recomputation, so a reader can
// pre-warm the state for the keys it is next-in-line to own.
func Replicas(key string, members []Member, n int) []Member {
	if n <= 0 || len(members) == 0 {
		return nil
	}
	type scored struct {
		m Member
		s uint64
	}
	all := make([]scored, len(members))
	for i, m := range members {
		all[i] = scored{m, weight(key, m.ID)}
	}
	for i := 1; i < len(all); i++ {
		for j := i; j > 0; j-- {
			a, b := all[j], all[j-1]
			less := a.s > b.s || (a.s == b.s && a.m.ID < b.m.ID)
			if !less {
				break
			}
			all[j], all[j-1] = all[j-1], all[j]
		}
	}
	if n > len(all) {
		n = len(all)
	}
	out := make([]Member, n)
	for i := 0; i < n; i++ {
		out[i] = all[i].m
	}
	return out
}

// weight is the HRW score for (key, member): the first 8 bytes of
// SHA-256(key || 0x00 || memberID) as a big-endian uint64. The NUL separator
// stops ("ab","c") and ("a","bc") from colliding.
func weight(key, memberID string) uint64 {
	h := sha256.New()
	_, _ = h.Write([]byte(key))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(memberID))
	sum := h.Sum(nil)
	return binary.BigEndian.Uint64(sum[:8])
}
