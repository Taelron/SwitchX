// Package config loads switchx's TOML configuration from the XDG config
// path, enforces the file-mode rule from the Taelron Security & Secret
// Handling baseline, and validates that all required fields are present.
//
// This package sits outside the four hexagonal layers (domain/app/storage/ui)
// because configuration loading is bootstrap-time only: it runs in
// cmd/switchx/main.go before any layer is wired.
//
// References (canonical in Linear):
//
//   - ADR-0004 — Configuration and Bootstrap
//     https://linear.app/taelron/document/adr-0004-configuration-and-bootstrap-e5000764e4c4
//   - ADR-0005 — Secret Handling for switchx
//     https://linear.app/taelron/document/adr-0005-secret-handling-for-switchx-ea21d3f379f4
//   - Security & Secret Handling (Taelron Baselines)
//     https://linear.app/taelron/document/security-and-secret-handling-75be68be36b6
package config
