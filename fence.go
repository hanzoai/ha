// Copyright 2025 Hanzo AI Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package ha

// Fencing is the ONE value election cannot supply on its own. Owner()/IsOwner()
// answer "who SHOULD write key K", coordination-free — which is exactly why two
// replicas holding different (stale) membership views can EACH elect themselves
// the writer for K (split-brain). Election has no way to make a deposed or
// partitioned writer STOP. A fencing token does: a monotone Round the elected
// owner stamps onto every write, which the STORE rejects once a later owner has
// advanced it (see github.com/hanzoai/vfs/replica.FencedStore).
//
// This package contributes only the composable VALUE (Lease) and the SEAM
// (Leases) — named for what it yields, not for what consumes it. It is NOT
// "Fencer": fencing is what the STORE does when it refuses a stale round, and an
// interface that issues a lease does not fence anything. It is NOT "Writer": it
// has no write method, and Writer already means something you write TO. It is NOT
// "Consensus": StaticLeases returns Round 1 from a single process and a CAS
// implementation consults one register — neither reaches agreement, and naming the
// seam after one possible source is the mistake this separation exists to avoid.
// It is NOT "Leader": Owner is per-KEY, so a thousand keys have a thousand owners
// and there is no single leader to name. It deliberately does NOT produce the round: a round is the output of
// a LINEARIZABLE source, and a linearizable source is coordination — folding it
// in would re-complect election with consensus, the precise separation this
// package exists to keep. Today that source is a single linearizable register
// (an object-store compare-and-set, or a coordination DB); tomorrow it is a
// BFT-agreed round (the Lux quasar RSM). Both plug in behind Leases with no
// change at any call site.
//
// The one invariant every implementation MUST uphold, on which all downstream
// safety rests:
//
//	The round is monotone non-decreasing per key across ALL callers, and STRICTLY
//	increases whenever the writer role changes hands. A newly-acquired lease
//	therefore always outranks every prior holder's, so the store can fence a
//	deposed writer simply by refusing any round below the highest it has admitted.
//
// Composition: Owner() picks who; Leases binds that who to a monotone round;
// the store admits a write iff its round is current. Each concern stays in its
// lane — this file adds a value and an interface, nothing that does I/O.

import "context"

// Round is a monotone fencing epoch for a key. It is the SOLE boundary value
// between coordination (this package) and storage (the fenced object store): the
// store knows only that rounds increase and that a lower round is stale — nothing
// about members, leases, elections, or how the round was decided.
type Round uint64

// Lease binds an elected Owner to the Round at which it holds the single-writer
// role for Key. It is a value, not a held lock: possession is never enforced by
// carrying a Lease but by the store admitting a write stamped with its Round. A
// Lease whose Round has been superseded is inert — the store rejects it — which
// is exactly what makes a still-running deposed writer harmless.
type Lease struct {
	Key   string // the ownership unit this lease fences (e.g. an org id).
	Owner Member // the elected writer that holds the role.
	Round Round  // the monotone round the owner stamps onto its writes.
}

// Leases issues the caller's current Lease for a key from a linearizable,
// monotone round source. It is the seam a consensus-backed round drops into with
// no change at call sites; the interim implementation reads and advances a round
// in a single linearizable store. This package defines the seam and the value
// only — never the source.
//
// A caller MUST fail CLOSED on an error: no Lease means no established round, and
// writing without a fresh round risks two writers at one epoch. A nil error is
// the only signal it is safe to write under.
type Leases interface {
	// Acquire returns the caller's Lease for key: itself as Owner, bound to the
	// round at which it currently holds the writer role. Implementations advance
	// the round when the role changes hands TO the caller and keep it on renewal,
	// so repeated Acquire calls by a stable owner return a stable round while a
	// takeover returns a strictly higher one. It returns an error when the caller
	// is not — or can no longer be — the writer for key; the caller then does not
	// write (and typically forwards to, or steps aside for, the true owner).
	Acquire(ctx context.Context, key string) (Lease, error)
}

// StaticLeases is the single-process Leases: the sole process is the sole
// writer, so every key is held by self at a fixed Round of 1 — exactly-once by
// construction, with no round source to consult. It is the correct Leases for
// local dev, a standalone binary, and tests, and the safe default when no
// linearizable source is wired — mirroring Static for Membership. A single
// process cannot have a deposed second writer, so a constant round is sound;
// the moment there are two writers, a real (linearizable) Leases is required.
func StaticLeases(self string) Leases { return staticLeases(self) }

type staticLeases string

func (s staticLeases) Acquire(_ context.Context, key string) (Lease, error) {
	return Lease{Key: key, Owner: Member{ID: string(s)}, Round: 1}, nil
}
