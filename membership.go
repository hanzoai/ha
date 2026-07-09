// Copyright 2025 Hanzo AI Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package ha

import "context"

// Membership is the seam election elects over: the live, writer-eligible replica
// set plus this replica's own stable ID. It is the ONE injectable source every
// consumer shares — production wires a cluster source (e.g. hanzoai/ha/k8s), tests
// and single-process mode wire Static.
//
// A consumer MUST fail CLOSED on a Members error or an empty set: election over an
// unknown or empty membership has no safe owner, so the caller skips rather than
// risk two writers. A nil error with a non-empty set is the only "proceed" signal.
type Membership interface {
	// Self is this replica's stable ID — the value HRW weights derive from. It must
	// be stable for the replica's life (e.g. its pod name / a persistent node id).
	Self() string
	// Members is the live writer-eligible set, or an error the caller fails closed
	// on. A nil error with a non-empty set is the only signal safe to proceed on.
	Members(ctx context.Context) ([]Member, error)
}

// Static is the single-process Membership: one member, itself, never an error — the
// sole process is the sole writer. It is the correct source for local dev, a
// standalone binary, and tests, and the safe default when no cluster source is
// wired (a single process is exactly-once by construction). id is this member's ID.
func Static(id string) Membership { return static(id) }

type static string

func (s static) Self() string { return string(s) }

func (s static) Members(context.Context) ([]Member, error) {
	return []Member{{ID: string(s)}}, nil
}
