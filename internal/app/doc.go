// Package app holds switchx's use cases (orchestration) and the repository
// port interfaces that storage adapters implement. It imports domain only;
// never storage, never ui, never driver libraries. Every use case takes
// context.Context as its first parameter.
//
// References (canonical in Linear):
//
//   - Hexagonal Architecture (Taelron Baselines)
//     https://linear.app/taelron/document/hexagonal-architecture-b142001f420e
//   - ADR-0003 — Adopt Hexagonal Architecture and Domain & Persistence Separation Baselines
//     https://linear.app/taelron/document/adr-0003-adopt-hexagonal-architecture-and-domain-and-persistence-3392ec24d042
package app
