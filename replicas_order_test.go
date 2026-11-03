// Copyright 2025 Hanzo AI Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package ha

import (
	"testing"
)

// TestReplicasRankingIsViewIndependent proves the whole failover ranking — not
// just the elected head — is a function of (key, member SET), never of the
// order a membership source happened to list the set in. A reader pre-warms
// state for the keys it is next in line to own: an order-dependent successor
// list is a wrong pre-warm, and two replicas disagreeing about the successor
// order is the same class of failure as two disagreeing about the owner.
func TestReplicasRankingIsViewIndependent(t *testing.T) {
	base := []Member{{ID: "r1"}, {ID: "r2"}, {ID: "r3"}, {ID: "r4"}, {ID: "r5"}}
	// Every rotation plus a reversal covers reordered views of the same set.
	views := [][]Member{base}
	for i := 1; i < len(base); i++ {
		v := append([]Member{}, base[i:]...)
		views = append(views, append(v, base[:i]...))
	}
	rev := make([]Member, len(base))
	for i, m := range base {
		rev[len(base)-1-i] = m
	}
	views = append(views, rev)

	keys := []string{"acme", "globex", "org/42", "shard-7", "nightly-job"}
	for _, key := range keys {
		want := Replicas(key, base, len(base))
		owner, _ := Owner(key, base)
		if want[0].ID != owner.ID {
			t.Fatalf("Replicas(%q)[0]=%q, want owner %q", key, want[0].ID, owner.ID)
		}
		for _, v := range views[1:] {
			got := Replicas(key, v, len(v))
			if len(got) != len(want) {
				t.Fatalf("Replicas(%q) length %d under reordered view, want %d", key, len(got), len(want))
			}
			for i := range want {
				if got[i].ID != want[i].ID {
					t.Fatalf("Replicas(%q)[%d]=%q under reordered view, want %q (full order must be view-independent)", key, i, got[i].ID, want[i].ID)
				}
			}
		}
	}
}

// TestWeightSeparatesKeyFromMember pins the NUL separator in the weight input:
// without it, ("ab","c") and ("a","bc") hash the SAME byte string and a key's
// ownership could be claimed by a member whose ID continues the key.
func TestWeightSeparatesKeyFromMember(t *testing.T) {
	if weight("ab", "c") == weight("a", "bc") {
		t.Fatal("weight(ab,c) == weight(a,bc): key and member are not separated in the hash input")
	}
	if weight("", "r1") != weight("", "r1") {
		t.Fatal("weight is not deterministic")
	}
}
