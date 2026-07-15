## Summary

<!-- What does this PR do, in a sentence or two? -->

## Why

<!-- What problem does it solve? Link the issue if one exists. -->

## Scope check

- [ ] This change implements **nothing** from `SCOPE.md`'s "Not V1" list
      *(or: this PR edits `SCOPE.md` first and makes the case for the promotion)*

## Testing

- [ ] `go build ./... && go vet ./... && go test ./...` pass locally
- [ ] Exercised against a live cluster (kind or otherwise) — describe what you did:

<!-- e.g. "scaled coredns 2→5→2 and watched the ticker + map update" -->

## ADR impact

- [ ] No architectural decision is changed
- [ ] …or a new/amended ADR in `docs/adr/` is included in this PR
