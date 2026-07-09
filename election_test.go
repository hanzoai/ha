// Copyright 2025 Hanzo AI Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package ha

import (
	"fmt"
	"testing"
)

// TestOwnerDeterministicAndExclusive proves the HRW election picks exactly one
// owner, order-independently, and every member agrees.
func TestOwnerDeterministicAndExclusive(t *testing.T) {
	members := []Member{{ID: "r3"}, {ID: "r1"}, {ID: "r2"}}
	reordered := []Member{{ID: "r1"}, {ID: "r2"}, {ID: "r3"}}

	o1, ok1 := Owner("acme", members)
	o2, ok2 := Owner("acme", reordered)
	if !ok1 || !ok2 || o1.ID != o2.ID {
		t.Fatalf("owner not order-independent: %q vs %q", o1.ID, o2.ID)
	}

	// Exactly one member is the owner across the set.
	owners := 0
	for _, m := range members {
		if IsOwner("acme", m.ID, members) {
			owners++
		}
	}
	if owners != 1 {
		t.Fatalf("IsOwner true for %d members, want exactly 1", owners)
	}

	// Empty membership fails closed (no owner).
	if _, ok := Owner("acme", nil); ok {
		t.Fatal("empty membership must have no owner")
	}

	// Distinct keys spread across replicas (not all pinned to one).
	seen := map[string]bool{}
	for _, key := range []string{"a", "b", "c", "d", "e", "f", "g", "h"} {
		o, _ := Owner(key, members)
		seen[o.ID] = true
	}
	if len(seen) < 2 {
		t.Fatalf("HRW pinned all keys to %d replica(s); want spread", len(seen))
	}
}

// TestReplicasOwnerFirstThenFailover proves Replicas ranks the owner first and
// returns an ordered, bounded successor list — the failover pre-warm order.
func TestReplicasOwnerFirstThenFailover(t *testing.T) {
	members := []Member{{ID: "r1"}, {ID: "r2"}, {ID: "r3"}, {ID: "r4"}}

	// The head of Replicas is exactly the elected Owner.
	owner, _ := Owner("acme", members)
	rep := Replicas("acme", members, 3)
	if len(rep) != 3 {
		t.Fatalf("Replicas returned %d, want 3", len(rep))
	}
	if rep[0].ID != owner.ID {
		t.Fatalf("Replicas[0]=%q, want owner %q", rep[0].ID, owner.ID)
	}

	// No duplicates, and n is clamped to the member count.
	if got := Replicas("acme", members, 99); len(got) != len(members) {
		t.Fatalf("Replicas clamp: got %d, want %d", len(got), len(members))
	}
	if got := Replicas("acme", members, 0); got != nil {
		t.Fatalf("Replicas(n=0)=%v, want nil", got)
	}
	if got := Replicas("acme", nil, 3); got != nil {
		t.Fatalf("Replicas(empty)=%v, want nil", got)
	}
}

// BenchmarkOwnerElection measures the per-write HRW single-writer check (the hot path
// gating every write when scaled to N replicas).
func BenchmarkOwnerElection(b *testing.B) {
	members := make([]Member, 16)
	for i := range members {
		members[i] = Member{ID: fmt.Sprintf("replica-%02d", i)}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = IsOwner("some-org-slug", "replica-07", members)
	}
}
