# ADR-0003: client-go informers over per-request Lists

**Date:** 2026-07-16
**Status:** Accepted
**Deciders:** Benjamin Glover

## Context

Through the walking-skeleton phase, `/api/topology` issued two List requests
(nodes, pods) against the API server on every browser poll — every 10 seconds,
regardless of whether anything had changed. Pod objects are 5–15KB of JSON
each, most of which kutopo discards after normalization. This is SNMP-era
polling: tolerable for one user against a kind cluster, rude against a large
shared cluster, and contrary to the project's stated success metric of **zero
steady-state API-server requests**. The Kubernetes API provides watch semantics
(List once, then a resumable event stream from that `resourceVersion`)
precisely so clients don't have to poll; client-go's informer machinery is the
canonical packaging of that pattern and the mechanism every controller in the
Kubernetes control plane is built on.

## Decision Drivers

- Success metric: zero API-server traffic when the cluster is idle.
- Freshness: the map should know about changes in milliseconds, not at the
  next poll boundary.
- Truthfulness (ADR-0002 extended to ourselves): if our cache goes stale, the
  UI must say so rather than serving old data as current.
- Portfolio: informers are the single most transferable client-go concept.
- Testability: cluster-state logic must run in CI without a cluster.

## Options Considered

### Option A: Keep per-request Lists (status quo)
**Pros:**
- Zero additional code; state is trivially always-fresh at request time.

**Cons:**
- O(cluster size) API-server load on every poll, forever, even when idle.
- Fails the project's own success metric.

### Option B: SharedInformerFactory + listers (chosen)
One initial List per resource, then a watch stream maintains an in-memory
mirror; `/api/topology` reads through listers. Watch errors are tracked and
surfaced as a `staleSince` field when the stream is degraded.

**Pros:**
- Steady-state cluster traffic is zero; changes land in the cache in
  milliseconds; endpoint responses are microsecond-scale RAM reads.
- Reconnect, resume, and re-List edge cases handled by client-go, not us.
- Snapshot assembly became a pure function, fully tested against fixtures and
  the fake clientset in CI.

**Cons:**
- Full node+pod cache held in memory (acceptable: it's a fraction of what any
  kubectl get -A already materializes).
- Startup gains a sync gate (~1s on kind; bounded by a 30s timeout with an
  RBAC-pointing error message).
- "Is the watch healthy?" has no direct client-go API; we infer degradation
  from watch-error recency (30s window), which can lag reality by up to that
  window.

### Option C: Hand-rolled List+Watch loop
**Pros:**
- No informer abstraction; full control over the reflector loop.

**Cons:**
- Reimplements resourceVersion resume, backoff, and re-List — the exact
  hard-won machinery informers exist to provide. All risk, no signal.

## Decision

**Chosen option: B.** It is the only option that meets the zero-steady-state
metric, and it converts the project's riskiest logic into pure functions with
CI coverage. The staleness-inference imprecision (up to 30s of optimism) is
accepted and mitigated by surfacing `staleSince` in both the payload and UI
banner the moment degradation is detected.

## Consequences

**Positive:**
- Verified live: response time ~0.4ms; a coredns scale 2→4→2 propagated
  create/terminate/delete events into the cache with no List traffic.
- The zombie-coredns scenario (ADR-0002) is now a regression test.

**Negative / Trade-offs:**
- Frontend data can be up to one browser-poll (10s) behind the cache; the
  cache itself is milliseconds behind the cluster. SSE (V2) removes the first
  gap; nothing removes the second, and nothing should.

**Risks:**
- Watch-health inference lags by up to 30s → revisit if client-go exposes a
  first-class stream-health signal.

## References

- ADR-0001 (architecture), ADR-0002 (truthfulness)
- Proposal v2.0 changelog, decision 3 (informers + browser polling)
- [Kubernetes API concepts: efficient detection of changes](https://kubernetes.io/docs/reference/using-api/api-concepts/#efficient-detection-of-changes)
