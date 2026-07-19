# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Read-only Go client for movies, theaters, showtimes, ticket prices, and seat
  availability from Caribbean Cinemas Puerto Rico.
- Embedded directory of all 31 theaters and their official route slugs.
- `cinemas` CLI commands for browsing the catalog, opening official checkout,
  and printing official checkout links.
- Version reporting through `cinemas --version`.
- Interactive terminal flow from movie discovery through official browser
  handoff.
- Configurable HTTP client, endpoint, site, circuit, and User-Agent options.

### Security

- Purchasing remains on the official Caribbean Cinemas website. The client does
  not accept credentials, hold seats, create orders, or process payments.

[Unreleased]: https://github.com/jasielrt/caribbeancinemas-go/commits/main
