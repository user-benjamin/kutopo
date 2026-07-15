# kutopo

**A node-centric topology map for Kubernetes clusters — one binary, your kubeconfig, a browser tab.**

kutopo draws your cluster the way SNMP tools drew networks: machines as boxes,
workloads as dots around them, everything color-coded by health. Not a resource
browser, not a workload-relationship graph — a spatial answer to *"what is
running on which node, and can I trust what I'm seeing?"*

> **Status: early development.** Pre-v0.1 — the walking skeleton works end-to-end
> against a live cluster, but expect sharp edges. Not yet distributed via krew.

<!-- TODO: screenshot of the topology view against a multi-node cluster -->

## Quickstart

Requires a kubeconfig whose user can `list` nodes and pods (read-only).

```sh
go build -o kutopo .
./kutopo                      # uses your current kubeconfig context
./kutopo --context my-cluster --port 8090
```

Then open <http://127.0.0.1:8090>. The map refreshes every 10 seconds. The
server binds localhost only — your kubeconfig's RBAC is the entire auth model,
exactly as with kubectl.

## The map doesn't lie to you

A pod's `phase` is written by its node's kubelet. When a node stops reporting,
its pods' phases freeze at the last report — a dead node happily shows a screen
full of "Running" pods forever. kutopo joins every pod against its node's
readiness and its `deletionTimestamp`:

- Pods on a non-reporting node render **gray with a dashed ring — "stale"** —
  because "we don't know" is information, and repeating a dead kubelet's last
  words is not.
- Pods marked for deletion render as **terminating**, distinct from healthy ones.

This rule exists because of a real incident, not a thought experiment — see
[ADR-0002](docs/adr/ADR-0002-join-pod-phase-with-node-readiness.md).

## How it works

A single Go binary: client-go reads nodes and pods through your kubeconfig,
normalizes them into a topology snapshot, and serves it at `/api/topology`
alongside a static UI (vis-network, vendored and compiled into the binary via
`go:embed` — no CDN, no build step, no runtime dependencies). Architecture
rationale, including the options that lost, lives in
[ADR-0001](docs/adr/ADR-0001-go-kubectl-plugin-with-local-browser-ui.md).

## Positioning

| Tool | What it shows | kutopo's difference |
|---|---|---|
| [KubeView](https://github.com/benc-uk/kubeview) | Workload relationships (Deployment → ReplicaSet → Pod → Service) | Node-centric: the *infrastructure* view |
| [Headlamp](https://headlamp.dev/) | General resource browser (SIG-UI recommended) | Purpose-built topology map, not a browser |
| [k9s](https://k9scli.io/) | Terminal tables and navigation | Complementary — k9s for hands, kutopo for eyes |
| Octant (archived 2023) | Local-binary + browser dashboard | The delivery model, revived, for topology |

## Roadmap

- **V1:** client-go informers replace per-request Lists (zero steady-state API
  server load), `kubectl-` plugin naming, multi-OS release binaries, kind e2e.
- **Post-V1:** krew distribution.
- **V2 (explicitly out of V1 scope):** SSE live push, namespace filtering,
  multi-cluster, in-cluster deployment mode with a real auth story.

## Design documents

Decisions are recorded as ADRs in [`docs/adr/`](docs/adr/).
