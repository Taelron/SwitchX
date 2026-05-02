// Package storage holds repository implementations (adapters) for the port
// interfaces defined in package app. Storage owns SQL, the connection pool,
// the migration runner, and driver-specific concerns. It imports domain and
// app (for the interfaces it implements); never ui. Domain types and row
// types are kept distinct, with mapper functions translating between them.
//
// References (canonical in Linear):
//
//   - Hexagonal Architecture (Taelron Baselines)
//     https://linear.app/taelron/document/hexagonal-architecture-b142001f420e
//   - Domain & Persistence Separation (Taelron Baselines)
//     https://linear.app/taelron/document/domain-and-persistence-separation-2df00a8622ca
//   - ADR-0002 — Postgres for Shared Team Storage
//     https://linear.app/taelron/document/adr-0002-postgres-for-shared-team-storage-b6b0df46161c
//   - ADR-0003 — Adopt Hexagonal Architecture and Domain & Persistence Separation Baselines
//     https://linear.app/taelron/document/adr-0003-adopt-hexagonal-architecture-and-domain-and-persistence-3392ec24d042
package storage
