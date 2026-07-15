# ADR-0002: Pod status must be joined with node readiness and deletionTimestamp

**Date:** 2026-07-14
**Status:** Accepted
**Deciders:** Benjamin Glover

## Context

On 2026-07-14, during development against the `cka-practice` kind cluster, a
Docker restart left the control-plane node's kubelet unable to reach the API
server for 24 hours (stale hardcoded IP in `kubelet.conf`). During that window
the cluster reported **four** coredns pods, all `phase=Running`, against a
deployment that wanted two. The two pods on the dead node were zombies: a pod's
`status.phase` is written by its node's kubelet, so when a kubelet stops
reporting, its pods' phases freeze at the last report — indefinitely. Meanwhile
the eviction machinery had already marked them for deletion (`deletionTimestamp`
set) and the ReplicaSet controller had built live replacements on healthy nodes.

A topology tool that renders `status.phase` verbatim would have drawn four green
pods — confidently displaying two pods that, for operational purposes, did not
exist. For a tool whose entire purpose is triage ("the map is trustworthy"),
this is the worst possible failure mode: it is most wrong precisely when the
cluster is most degraded.

## Decision Drivers

- The product promise is a *trustworthy* map for triaging degraded clusters —
  the tool must be most accurate exactly when nodes are failing.
- Kubernetes reports state truthfully but distributed-ly: correctness requires
  joining facts that no single API field contains.
- V1 simplicity: the fix must not require additional API calls or permissions
  beyond the existing node/pod reads.

## Options Considered

### Option A: Render `status.phase` verbatim
The naive implementation — one field, no joins.

**Pros:**
- Simplest possible code; matches what `kubectl get pods` shows.

**Cons:**
- Provably wrong during node failures (observed live, not hypothetical).
- Silently over-reports healthy workloads during the exact scenarios the tool
  exists to triage.

### Option B: Join phase with node readiness and deletionTimestamp (chosen)
Server-side, per pod: if the pod's node exists and is NotReady, mark the pod
`stale` (rendered gray with a dashed ring — explicitly "unknown", not a status
assertion); if `deletionTimestamp` is set, mark it `terminating`. Precedence:
stale > terminating > phase.

**Pros:**
- Uses only data already fetched (nodes + pods) — no new calls or RBAC.
- Renders epistemic honesty: "we don't know" is displayed as not-knowing rather
  than repeating a dead kubelet's last words.
- The derived fields live in the API payload, keeping the frontend dumb.

**Cons:**
- "Stale" is not a Kubernetes API state; users must learn one tool-specific concept.
- A brief node flap grays out its pods until readiness recovers.

### Option C: Independently verify pod liveness (probe endpoints, query metrics)
Cross-check phase against a second source of truth.

**Pros:**
- Highest-fidelity answer to "is this pod actually alive?"

**Cons:**
- Requires network reachability to pods, extra permissions, and per-pod probes —
  a different product (a health checker, not a topology map).
- Massive V1 scope increase for marginal gain over Option B's honest "unknown".

## Decision

**Chosen option: B — join with node readiness and deletionTimestamp.**

Option B satisfies the trustworthiness driver at zero additional API cost:
node readiness is already in the snapshot, and the join is a map lookup. Option
A was disqualified by direct observation. Option C answers a question the tool
does not ask — kutopo's job is to render the control plane's knowledge honestly,
including its gaps, not to build an independent monitoring system.

## Consequences

**Positive:**
- During node failures the map degrades honestly: affected pods turn gray-dashed
  ("node not reporting") instead of remaining confidently green.
- Zombie pods (deleted-but-unconfirmed) are visually distinct from healthy ones.

**Negative / Trade-offs:**
- One tool-specific status ("Stale") to document in the README and legend.
- Node-readiness flaps briefly gray out healthy pods (accepted; a grace period
  can be added if it proves noisy in practice).

**Risks:**
- Future resource types (e.g., DaemonSet-managed static pods) may need refined
  join rules — revisit when the resource surface grows.

## References

- Incident diagnosis session, 2026-07-14 (kind IP-reshuffle → kubelet lockout)
- [Kubernetes node conditions & heartbeats](https://kubernetes.io/docs/concepts/architecture/nodes/#condition)
- ADR-0001 (core architecture)
