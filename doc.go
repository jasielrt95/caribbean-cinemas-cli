// Package caribbeancinemas is a read-only Go client for the Caribbean Cinemas
// (Puerto Rico) GraphQL API, which runs on the INDY Cinema Group / Fandango
// platform.
//
// The read side of the API is unauthenticated: no cookies, session, or API key
// are required. This library wraps only read operations — browsing movies,
// theaters, showtimes, seat maps, and pricing. It deliberately does NOT support
// ordering, seat holds, payments, or any authenticated/write operation; those
// live behind a customer login and are out of scope.
//
// # Quick start
//
//	client := caribbeancinemas.New()
//	movies, err := client.NowPlaying(ctx, caribbeancinemas.AllSites())
//
// # Data model note
//
// A movie does not have one global ID. Each (movie × theater) pair is its own
// record with its own ID, and a movie's showtimes are scoped to the theater
// that ID belongs to. See Client.MoviesAtSite and Client.Showtimes.
//
// # CORS / handoff
//
// The GraphQL endpoint is same-origin only, so a browser cannot call it
// directly — this library is meant to run server-side. For purchasing, which
// the API does not expose, use the Deeplink helpers to hand users off to the
// official Caribbean Cinemas checkout.
package caribbeancinemas
