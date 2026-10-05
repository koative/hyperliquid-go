<!--
The PR title becomes the squash-merge commit message and must follow
Conventional Commits (feat:, fix:, docs:, refactor:, test:, chore:, ci:).
Mark breaking changes with "!" (feat!: …) and explain them below.
-->

## What and why

## How it was tested

<!-- New or changed exchange actions: say how the signature was verified (golden vector or Testnet). -->

- [ ] `go test -race ./...` and `golangci-lint run` pass
- [ ] Exported identifiers have doc comments
