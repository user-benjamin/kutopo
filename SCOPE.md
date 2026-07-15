# SCOPE.md

This file is a commitment device, not documentation. The project proposal
identified scope creep as a high-impact risk and named this file as the
mitigation: **any change that implements something from the "not V1" lists
below is declined by pointing at this file**, however good it feels in the
moment. Features get promoted by editing this file first, deliberately — never
by a PR that "just adds it while we're in there."

## V1 — what ships

- Node-centric topology map: nodes and pods, status-colored, browser UI served
  from a single embedded binary (done — walking skeleton)
- Truthfulness join: stale / terminating derived states per ADR-0002 (done)
- Detail panels: pod restarts, ready ratio, images, owner, age; node version,
  IP, allocatable, pressures (done)
- client-go **informers** replacing per-request Lists — zero steady-state API
  server load (the remaining V1 build milestone)
- `kubectl-` plugin binary naming, goreleaser multi-OS release binaries
- Edge cases: empty cluster, RBAC-denied startup, unreachable cluster banner
- README with screenshot, `docs/rbac.yaml` sample

## Parked — agreed shape, V1-legal, build only after informers land

- Double-click a pod → copyable `kubectl exec` / `logs` / `describe` commands
  in the panel (reserves the gesture the V2 terminal will inherit)
- Namespace highlight strip in the header (emphasis, not filtering)

## Not V1 — declined by reference to this file

- **Browser-based exec terminal** (xterm.js + WebSocket + exec subresource).
  Not deferred for difficulty — deferred because it converts a read-only
  localhost server into a command proxy for everything the kubeconfig can
  touch, which demands its own security design ADR (origin checks, session
  token, possibly opt-in flag) before a line of it is written.
- SSE live push (V1's 10s localhost polling is perceptually equivalent for triage)
- Namespace filtering / selector UI
- Multi-cluster support (V1: one context via `--context`)
- In-cluster deployment mode (returns only with a real auth story)
- Metrics, logs, events, or any second data source beyond nodes + pods

## Post-V1 rollout (not features)

- krew index submission
