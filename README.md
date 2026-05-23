# switchx

Terminal-native time tracker for consulting teams.

## Status

Phase 1 — implementation in progress. **M1 (Foundation & Bootstrap)
closed.** See [`docs/m1-smoke-test.md`](docs/m1-smoke-test.md) for the
end-to-end verification procedure.

## First-run setup

```
cp .local/.env.example .local/.env   # template → real values
$EDITOR .local/.env                  # fill in AZURE_VAULT (at minimum)
make dev-up                          # bootstrap local PG role + database
```

## Documentation

The canonical documentation lives in Linear:

- [switchx project](https://linear.app/taelron/project/switchx-0b0069bd1c04)
- [README (full)](https://linear.app/taelron/document/readme-3b00ab0b67b2)
- [Specification](https://linear.app/taelron/document/specification-3645653a2170)
- [Domain Model](https://linear.app/taelron/document/domain-model-11af391c0064)

## License

Apache 2.0. See `LICENSE`.
