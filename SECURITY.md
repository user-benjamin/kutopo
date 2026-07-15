# Security Policy

kutopo V1 is a **read-only, localhost-only** tool: it binds `127.0.0.1`,
serves your kubeconfig's view of nodes and pods, and has no write path to the
cluster. That's a deliberately small attack surface — but if you find a way
through it (or a way to make kutopo misrepresent cluster state), please report
it privately.

**How to report:** use GitHub's private vulnerability reporting
(*Security → Report a vulnerability* on this repo). Please don't open a public
issue for suspected vulnerabilities.

**Supported versions:** the latest release and `main`.
