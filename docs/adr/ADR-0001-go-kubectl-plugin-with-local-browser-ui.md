# ADR-0001: Go kubectl plugin serving a local browser UI

**Date:** 2026-07-14 (decision made 2026-07-13)
**Status:** Accepted
**Deciders:** Benjamin Glover

## Context

kutopo is a node-centric Kubernetes topology visualizer: an SNMPc-style spatial
map of which pods run on which nodes, color-coded by health. The original
proposal (v1.0) specified a Python/FastAPI web dashboard deployed in-cluster.
A landscape review (2026-07-13) changed the picture: the official Kubernetes
Dashboard is archived and Headlamp is the SIG-UI-recommended general resource
browser; KubeView (Go, actively maintained) covers workload-relationship graphs;
Octant — which pioneered the local-binary-plus-browser model — was archived in
January 2023 and never replaced. The unoccupied niche is node-centric topology
delivered with kubectl-native ergonomics. Separately, the in-cluster deployment
model forced an unsolved authentication problem (the v1.0 proposal punted it to
"trusted network"), and the `pip install` path contradicted the project's
zero-dependency goal.

## Decision Drivers

- Portfolio signal: the project must demonstrate platform-engineering skills in
  the ecosystem's own idiom (Go, client-go, kubectl plugin conventions).
- True zero-dependency install: one artifact, no runtime, no CDN, no build step.
- No authentication surface: the tool must not need its own login/session layer.
- A spatial topology graph requires pixel rendering — terminals cannot draw it.
- Solo, part-time development: distribution and packaging overhead must be minimal.

## Options Considered

### Option A: Python/FastAPI in-cluster dashboard (original proposal)
Deployed as a pod with a ServiceAccount, UI served over the cluster network.

**Pros:**
- Familiar stack; fastest initial development.
- Always-on: the dashboard exists independently of any operator's machine.

**Cons:**
- Unsolved auth story — anyone who can reach the Service sees the cluster.
- "Zero-dependency" is false: requires Python runtime, pip, and a deploy manifest.
- Weak hiring signal: the Python client is a wrapper; the ecosystem is Go.
- Competes head-on with KubeView's deployment model.

### Option B: Go binary as a kubectl plugin, serving a localhost browser UI (chosen)
A static binary named for kubectl plugin discovery; loads the user's kubeconfig,
serves `127.0.0.1` only, UI and vendored JS embedded via `go:embed`.

**Pros:**
- Kubeconfig RBAC *is* the auth layer — identical trust model to kubectl itself.
- Single artifact per OS; `go:embed` makes the zero-dependency claim literally true.
- client-go, informers, and krew distribution are the strongest available signals.
- Browser rendering keeps the mature JS graph-visualization ecosystem (vis-network).

**Cons:**
- Not always-on: the map exists only while the operator runs the tool.
- client-go is verbose and has a real learning curve (+1 week budgeted).
- In-cluster mode becomes a V2 feature rather than the default.

### Option C: Native client — desktop app (Wails/Fyne) or TUI (Bubble Tea)
A windowed application or a k9s-style terminal UI.

**Pros:**
- Desktop: dock presence, potential for notifications and daily-driver use.
- TUI: trivially distributable, at home in the target audience's terminal.

**Cons:**
- Wails renders the same HTML/JS in an OS webview — identical frontend code with
  an added per-OS signing/notarization tax (macOS Gatekeeper blocks unsigned apps).
- Native Go GUI toolkits have essentially no graph-visualization ecosystem;
  layout, zoom/pan, and hit-testing would be hand-rolled.
- A TUI cannot draw a spatial topology map — it abandons the core value
  proposition (k9s wins at tables precisely because it does not attempt maps).

## Decision

**Chosen option: B — Go kubectl plugin with local browser UI.**

Option B is the only option that satisfies all five drivers simultaneously: it
deletes the authentication problem rather than mitigating it (driver 3), makes
the zero-dependency claim true via a single embedded binary (driver 2), keeps
browser-grade rendering for the topology map (driver 4), and places the project
squarely in the hiring-signal idiom (driver 1) — at the accepted cost of not
being always-on and a steeper client-go learning curve. Data freshness follows
the same philosophy: client-go informers (watch-based) on the cluster hop, with
simple 10-second polling on the free localhost hop; SSE push is deferred.

## Consequences

**Positive:**
- No login, session, or `0.0.0.0` exposure questions exist anywhere in V1.
- Install is "download one file"; krew becomes a distribution channel post-V1.
- The informer layer doubles as the project's primary learning objective.

**Negative / Trade-offs:**
- No persistent dashboard; in-cluster mode (with a real auth story) is deferred to V2.
- Browser-tab UX depends on the operator's default browser; no window management.
- Multi-OS release automation (goreleaser) is required for credible distribution.

**Risks:**
- client-go verbosity blows estimates → +1 week already budgeted; fake-clientset
  tests allow progress without a live cluster.
- vis-network layout degrades on large clusters → physics caps and a documented
  node-count limit in V1.

## References

- Project proposal v2.0 (decision changelog): `~/workspace/project-proposals/03-k8s-topo.md`
- Prior art: [KubeView](https://github.com/benc-uk/kubeview),
  [Headlamp](https://kubernetes.io/blog/2026/06/25/headlamp-cluster-api-plugin/),
  [Octant (archived)](https://github.com/vmware-archive/octant)
- ADR-0002 (topology truthfulness)
