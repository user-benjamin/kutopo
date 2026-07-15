# Contributing to kutopo

Thanks for your interest. kutopo is a small, deliberately scoped tool — the
most useful thing you can read before contributing is [`SCOPE.md`](SCOPE.md).

## The scope rule (read this first)

kutopo's V1 boundary is a commitment, not a suggestion. PRs that implement
anything on SCOPE.md's **"Not V1"** list will be declined by reference to that
file, regardless of quality — it's nothing personal, it's the project's core
discipline. If you believe something deserves promotion, open an issue that
argues for editing SCOPE.md itself; features move by changing the scope file
first, never by a PR that "just adds it."

For anything non-trivial, open an issue before writing code.

## Development setup

You need Go (see `go.mod` for the version) and a cluster to point at —
[kind](https://kind.sigs.k8s.io/) works well:

```sh
kind create cluster
go build -o kutopo .
./kutopo            # serves http://127.0.0.1:8090 against your current context
```

**The one gotcha:** the frontend (`static/`) is compiled into the binary via
`go:embed`. Editing `static/index.html` does nothing to a running server —
rebuild and restart to see UI changes.

## Testing

```sh
go vet ./...
go test ./...
```

Logic that touches the Kubernetes API should be testable against the client-go
fake clientset (`k8s.io/client-go/kubernetes/fake`) — no live cluster required.

## Style

- Go: `gofmt`-clean, `go vet`-clean, standard library first. There is no web
  framework and there isn't going to be one for two routes.
- Frontend: deliberately buildless — vanilla JS, no npm, no bundler, no
  toolchain. vis-network stays vendored in `static/`.
- Color and status rendering follow the rules in ADR-0002: a pod on a
  non-reporting node is *unknown*, never green.

## Decisions

Significant design choices are recorded in [`docs/adr/`](docs/adr/). If your
change alters an architectural decision, it needs a new ADR (or an amendment)
in the same PR.

## Branches and history

Short-lived branches off `main`, small focused PRs, linear history (squash or
rebase — merge commits are disabled). CI must be green.

## License

By contributing, you agree your contributions are licensed under Apache-2.0.
