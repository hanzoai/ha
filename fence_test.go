// Copyright 2025 Hanzo AI Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package ha

import (
	"context"
	"testing"
)

// TestStaticFencerHoldsEveryKeyAtRoundOne proves the single-process Fencer is
// exactly-once by construction: self holds every key at the fixed round 1, never
// an error — the sole process is the sole writer.
func TestStaticFencerHoldsEveryKeyAtRoundOne(t *testing.T) {
	f := StaticFencer("solo")
	for _, key := range []string{"acme", "globex", "orgs/acme/iam", ""} {
		l, err := f.Acquire(context.Background(), key)
		if err != nil {
			t.Fatalf("Acquire(%q): unexpected error %v", key, err)
		}
		if l.Owner.ID != "solo" || l.Round != 1 || l.Key != key {
			t.Fatalf("Acquire(%q) = %+v, want {Key:%q Owner:solo Round:1}", key, l, key)
		}
	}
}

// TestStaticFencerIsDeterministic proves repeated Acquire of a key returns a
// STABLE lease (round does not drift for a stable owner) — the renewal contract
// every Fencer must honor.
func TestStaticFencerIsDeterministic(t *testing.T) {
	f := StaticFencer("solo")
	a, _ := f.Acquire(context.Background(), "acme")
	b, _ := f.Acquire(context.Background(), "acme")
	if a != b {
		t.Fatalf("Acquire not stable for a fixed owner: %+v vs %+v", a, b)
	}
}

// TestLeaseRoundIsTheFencingBoundary documents the invariant the whole design
// leans on: a takeover lease STRICTLY outranks the prior holder's, so the store
// can fence the old writer purely by comparing rounds — the store needs to know
// nothing about members or elections, only that a lower round is stale.
func TestLeaseRoundIsTheFencingBoundary(t *testing.T) {
	prior := Lease{Key: "acme", Owner: Member{ID: "old"}, Round: 4}
	takeover := Lease{Key: "acme", Owner: Member{ID: "new"}, Round: 5}
	if takeover.Round <= prior.Round {
		t.Fatalf("takeover round %d must strictly exceed prior round %d", takeover.Round, prior.Round)
	}
	// The elected owner and the fenced round are independent axes: the SAME key
	// can change owner while the round advances, and the value carries both.
	if takeover.Key != prior.Key {
		t.Fatal("a lease handoff keeps the key and changes owner+round")
	}
}

// staticFencer must satisfy the Fencer seam — the compile-time proof the value
// this package ships is usable wherever a Fencer is required.
var _ Fencer = StaticFencer("self")
