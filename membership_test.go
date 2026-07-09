// Copyright 2025 Hanzo AI Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package ha

import (
	"context"
	"testing"
)

// TestStaticIsSelfOwner proves single-process Membership always elects itself: the
// sole member owns every key, so a standalone binary is exactly-once by construction.
func TestStaticIsSelfOwner(t *testing.T) {
	m := Static("solo")
	if m.Self() != "solo" {
		t.Fatalf("Self()=%q, want solo", m.Self())
	}
	members, err := m.Members(context.Background())
	if err != nil {
		t.Fatalf("Members: %v", err)
	}
	if len(members) != 1 || members[0].ID != "solo" {
		t.Fatalf("Members=%v, want [{solo}]", members)
	}
	for _, key := range []string{"a", "b", "c"} {
		if !IsOwner(key, m.Self(), members) {
			t.Fatalf("static member must own every key; missed %q", key)
		}
	}
}
