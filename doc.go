// Copyright 2025 Hanzo AI Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

// Package ha is Hanzo's high-availability coordination primitive: single-writer
// election over a live membership set, with no coordinator, no lock service, no
// Postgres, no Redis. It answers one question identically on every replica —
// "which replica is the single writer for key K right now?" — via Rendezvous
// (Highest-Random-Weight) hashing, so N stateless replicas agree on one owner per
// key and fail over deterministically with no recomputation.
//
// ha decides WHO writes; it never touches HOW state is stored or replicated. That
// separation is the point: compose ha with hanzoai/vfs (per-org SQLite + object
// store) for HA storage, or with a cron tick, a queue consumer, or any singleton
// that must run exactly once across a multi-replica Deployment. The same election
// guards all of them.
//
// The seam is Membership: election is a pure function of (key, []Member), and a
// Membership supplies the live set. Wire a cluster source (hanzoai/ha/k8s) in
// production, or Static for a single process and tests.
package ha
