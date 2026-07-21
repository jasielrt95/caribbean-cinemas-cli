# Contributing

Thanks for helping improve Caribbean Cinemas CLI.

## Development

Use Go 1.26.5 or newer. From the repository root, run:

```sh
go mod download
go test -race -count=1 ./...
go vet ./...
go run golang.org/x/vuln/cmd/govulncheck@v1.6.0 ./...
go build ./...
test -z "$(gofmt -l .)"
```

Keep the public library and CLI read-only. Purchasing must remain an explicit
handoff to the official Caribbean Cinemas website; do not add ordering, seat
holds, authentication, payment handling, or checkout automation.

When the undocumented GraphQL API changes, update tests and public documentation
together. Avoid live tests that reserve inventory or mutate production data.

## Pull requests

- Keep changes focused and explain user-visible behavior.
- Add or update tests for behavior changes.
- Update `README.md` or `CHANGELOG.md` when appropriate.
- Ensure the validation commands above pass before requesting review.
