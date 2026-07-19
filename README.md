# caribbeancinemas-go

A read-only Go client and CLI for [Caribbean Cinemas](https://home.caribbeancinemas.com)
(Puerto Rico). It wraps the theater chain's public GraphQL API to browse movies,
theaters, showtimes, seat availability, and prices — then hands users off to the
official site to buy.

The read API needs no authentication. This library **only reads**: it never
places orders, holds seats, or takes payments (those require a customer login
and are out of scope). For purchasing, it builds deep links into the official
web app.

## Requirements

- Go 1.26.5 or newer
- macOS, Linux, or Windows

## Install the CLI

Once the repository is published, install the latest release with:

```sh
go install github.com/jasielrt/caribbeancinemas-go/cmd/cinemas@latest
```

Go installs the executable into `GOBIN`, or into `GOPATH/bin` when `GOBIN` is
unset. If `cinemas` is not found after installation, add that directory to your
shell path. For the default Go configuration on macOS or Linux:

```sh
export PATH="$(go env GOPATH)/bin:$PATH"
```

To make that permanent in zsh, add the same line to `~/.zshrc`, then open a new
terminal. On Windows, add `<GOPATH>\bin` to your user `Path`, replacing
`<GOPATH>` with the value printed by `go env GOPATH`.

Confirm the installation with:

```sh
cinemas --help
cinemas --version
```

### Install from a local checkout

From the repository root:

```sh
go install ./cmd/cinemas
cinemas --help
cinemas --version
```

### Use the Go library

From an existing Go module:

```sh
go get github.com/jasielrt/caribbeancinemas-go@latest
```

## Library quick start

```go
client := caribbeancinemas.New()
ctx := context.Background()

// Movies at Plaza Las Américas (site 45). Each movie's ID is specific to
// this theater — that's what you pass to Showtimes.
movies, _ := client.MoviesAtSite(ctx, "45")

showings, _ := client.Showtimes(ctx, movies[0].ID)
sheet, _   := client.Pricing(ctx, showings[0].ID)   // Adult $7.18, Children $4.93, ...
chart, _   := client.SeatChartForShowing(ctx, showings[0].ID)
fmt.Println(len(chart.AvailableSeats()), "seats open")

// Hand off to the official site for seat selection + checkout.
site, _ := caribbeancinemas.TheaterByID(movies[0].SiteID)
buyURL := caribbeancinemas.NewDeeplinker("").SeatSelectionURL(site.Slug, showings[0].ID)
```

See [`examples/basic`](examples/basic) for the full flow.

## CLI

The flagship is a full-screen TUI that walks the whole flow:

```sh
cinemas interactive          # movie -> theater -> showtime -> open official checkout
```

Or the individual commands:

```sh
cinemas theaters                              # all 31 theaters
cinemas movies                                # now playing across the circuit (grouped)
cinemas movies --site 45                      # at one theater (site-specific IDs)
cinemas showtimes --title moana --site 45     # upcoming showtimes
cinemas showtimes --movie-id 485824           # or pass a site-specific movie ID
cinemas price --showing 882869                 # ticket prices
cinemas seats --showing 882869                 # ASCII seat map with availability
cinemas checkout --showing 882869              # open official seat selection
cinemas buy-link --showing 882869              # print the official URL
```

## The one thing to know: per-theater movie IDs

A movie does **not** have a single global ID. Each *(movie × theater)* pair is
its own record with its own ID, and a movie's showtimes are scoped to the
theater that ID belongs to.

- To list a movie across theaters, call `ListMovies` with many site IDs and use
  `GroupByTitle` to answer "which cinemas show this film?".
- To get showtimes at a specific theater, use that theater's site-specific movie
  ID (from `MoviesAtSite` / `ListMovies`), then `Showtimes`.

## What it can and can't do

| Capability | Supported |
|------------|-----------|
| Browse movies (now playing, coming soon, future/events) | Yes |
| Theater directory (names, addresses, some coordinates) — embedded, no network | Yes |
| Showtimes, incl. per-showing format (dubbed/subtitled/premium) | Yes |
| Ticket prices per showing | When exposed by the public API |
| Seat map + availability (point-in-time snapshot) | Yes |
| Purchase handoff link (seat selection / checkout) | Yes |
| Buying, seat holds, live seat locking, accounts, gift cards | No (auth-gated) |

## Purchase handoff

The interactive flow deliberately stops at the official-site boundary. After
choosing a movie, theater, and showtime, press `o` to open Caribbean Cinemas in
your normal browser. Seat selection, login, food, payment, and order management
remain entirely on the official site.

Your app owns discovery and browsing; the official site owns checkout. Carry the
user from movie → theater → showtime, then open
`Deeplinker.SeatSelectionURL(siteSlug, showingID)` — ideally in an in-app
browser. It lands the user **directly on the seat map** for that exact showtime,
where they continue as guest (no login), pick seats, and pay on Caribbean
Cinemas' own site. No credentials or payment data ever touch your app.

```go
site, _ := caribbeancinemas.TheaterByID(movie.SiteID)
url := caribbeancinemas.NewDeeplinker("").SeatSelectionURL(site.Slug, showing.ID)
// https://home.caribbeancinemas.com/plaza-americas/checkout/seats/879144
```

**Route notes.** The web app's routes are theater-slug prefixed
(`/{siteSlug}/checkout/seats/{showingId}`). The theater slug comes from
`Site.Slug` and is **not** derivable from the name (`The Outlet 66` → `belz`),
so use the embedded value. The unscoped `/movie/{slug}` and `/seats/{id}` routes
redirect to a location picker — always use the slug-prefixed form. Verified
against the live site.

## Design notes

- Standard library only for the core client (`net/http`, `encoding/json`);
  Cobra and Bubble Tea power the CLI and interactive terminal flow.
- Configurable via options: `WithHTTPClient`, `WithSiteID`, `WithUserAgent`, etc.
- All API methods take a `context.Context`.
- Errors from the API surface as `*APIError` (with a `Code`); use
  `IsAuthRequired` to detect login-gated queries.

## License

[MIT](LICENSE)
