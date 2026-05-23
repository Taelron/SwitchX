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
make dev-up                          # provisions PG + writes .local/dev-config/
make run                             # runs against the dev config — ~/.config untouched
```

The dev binary launched via `make` always reads from `.local/dev-config/`,
which is separate from `~/.config/switchx/` (where a deployed `switchx`
installed via `go install` reads its real user data). The two configs
do not interact.

See [`CONTRIBUTING.md`](CONTRIBUTING.md) for the full env-var contract
and where defaults live (scripts vs. Makefile).

## Documentation

The canonical documentation lives in Linear:

- [switchx project](https://linear.app/taelron/project/switchx-0b0069bd1c04)
- [README (full)](https://linear.app/taelron/document/readme-3b00ab0b67b2)
- [Specification](https://linear.app/taelron/document/specification-3645653a2170)
- [Domain Model](https://linear.app/taelron/document/domain-model-11af391c0064)

## License

Apache 2.0. See `LICENSE`.
