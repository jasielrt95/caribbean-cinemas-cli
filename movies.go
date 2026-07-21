package caribbeancinemas

import (
	"context"
	"fmt"
	"sync"
)

// Movie list "type" values accepted by the movies query.
const (
	TypeNowPlaying = "now-playing"
	TypeComingSoon = "coming-soon"
	TypeFuture     = "future"
)

// Title class IDs, confirmed via the titleClasses query. Pass to
// MovieListOptions.TitleClassIDs to filter by content category.
const (
	ClassFilm          = "973444"
	ClassSpecialEvents = "973445"
	ClassNewThisWeek   = "973450"
	ClassOpera         = "973452"
	ClassBallet        = "973455"
	ClassPlay          = "973456"
)

// DefaultTitleClasses is the set the public website requests for the standard
// "now playing" grid (films plus alternative content).
var DefaultTitleClasses = []string{
	ClassFilm, ClassSpecialEvents, ClassNewThisWeek, ClassOpera, ClassBallet, ClassPlay,
}

const moviesQuery = `query ($limit: Int, $type: String, $subtype: String, $siteIds: [ID], $orderBy: String, $titleClassIds: [ID]) {
  movies(limit: $limit, type: $type, subtype: $subtype, siteIds: $siteIds, orderBy: $orderBy, titleClassIds: $titleClassIds) {
    data {
      id name abbreviation urlSlug showingStatus synopsis starring directedBy producedBy writers
      duration genre allGenres rating ratingReason originalLanguage countryOfOrigin
      trailerYoutubeId posterImage bannerImage color releaseDate dateOfFirstShowing
      datesWithPublicShowing showingCount isMarathon siteId titleClassId
    }
    count
  }
}`

// MovieListOptions controls a movie listing. The zero value lists now-playing
// films across the given sites. SiteIDs is required (use AllSites for the whole
// circuit).
type MovieListOptions struct {
	SiteIDs       []string
	Type          string   // TypeNowPlaying (default), TypeComingSoon, TypeFuture
	TitleClassIDs []string // defaults to DefaultTitleClasses
	Limit         int      // defaults to 100
}

// ListMovies returns movies matching opts. Because IDs are per-theater, the same
// title appears once per site it plays at (distinguished by Movie.SiteID). Use
// GroupByTitle to collapse them.
func (c *Client) ListMovies(ctx context.Context, opts MovieListOptions) ([]Movie, error) {
	if len(opts.SiteIDs) == 0 {
		return nil, fmt.Errorf("caribbeancinemas: ListMovies requires at least one site ID")
	}
	typ := opts.Type
	if typ == "" {
		typ = TypeNowPlaying
	}
	classes := opts.TitleClassIDs
	if len(classes) == 0 {
		classes = DefaultTitleClasses
	}
	limit := opts.Limit
	if limit == 0 {
		limit = 100
	}
	vars := map[string]any{
		"limit":         limit,
		"type":          typ,
		"subtype":       "watched",
		"siteIds":       opts.SiteIDs,
		"orderBy":       "magic",
		"titleClassIds": classes,
	}
	var resp struct {
		Movies struct {
			Data  []Movie `json:"data"`
			Count int     `json:"count"`
		} `json:"movies"`
	}
	if err := c.execute(ctx, "", moviesQuery, vars, &resp); err != nil {
		return nil, err
	}
	return resp.Movies.Data, nil
}

// NowPlaying is a convenience wrapper for the current films at the given sites.
func (c *Client) NowPlaying(ctx context.Context, siteIDs []string) ([]Movie, error) {
	return c.ListMovies(ctx, MovieListOptions{SiteIDs: siteIDs, Type: TypeNowPlaying})
}

// MoviesAtSite lists now-playing movies at a single theater. The returned
// Movie.ID values are the site-specific IDs needed by Showtimes.
func (c *Client) MoviesAtSite(ctx context.Context, siteID string) ([]Movie, error) {
	return c.ListMovies(ctx, MovieListOptions{SiteIDs: []string{siteID}})
}

// NowPlayingEverywhere fetches now-playing movies at each of the given sites
// concurrently and returns the merged list, every Movie tagged with its SiteID.
//
// Unlike ListMovies with many site IDs (which collapses each title to a single
// representative theater), this returns one record per (movie × theater), so
// GroupByTitle over the result correctly reports every theater showing a film.
// Individual site failures are tolerated; an error is returned only if no site
// could be reached.
func (c *Client) NowPlayingEverywhere(ctx context.Context, siteIDs []string) ([]Movie, error) {
	const maxConcurrent = 8
	sem := make(chan struct{}, maxConcurrent)
	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		all      []Movie
		firstErr error
	)
	for _, site := range siteIDs {
		wg.Add(1)
		go func(site string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			movies, err := c.MoviesAtSite(ctx, site)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				if firstErr == nil {
					firstErr = err
				}
				return
			}
			all = append(all, movies...)
		}(site)
	}
	wg.Wait()

	if len(all) == 0 && firstErr != nil {
		return nil, firstErr
	}
	return all, nil
}

const movieByIDQuery = `query ($id: ID!) {
  movie(id: $id) {
    id name abbreviation urlSlug showingStatus synopsis starring directedBy producedBy writers
    duration genre allGenres rating ratingReason originalLanguage countryOfOrigin
    trailerYoutubeId posterImage bannerImage color releaseDate dateOfFirstShowing
    datesWithPublicShowing showingCount isMarathon siteId titleClassId
  }
}`

// MovieByID fetches a single movie by its (site-specific) ID. Returns ErrNotFound
// if no movie has that ID.
func (c *Client) MovieByID(ctx context.Context, id string) (*Movie, error) {
	vars := map[string]any{"id": id}
	var resp struct {
		Movie *Movie `json:"movie"`
	}
	if err := c.execute(ctx, "", movieByIDQuery, vars, &resp); err != nil {
		return nil, err
	}
	if resp.Movie == nil {
		return nil, ErrNotFound
	}
	return resp.Movie, nil
}

// TitleGroup is a single film collapsed across the theaters that show it.
type TitleGroup struct {
	Name    string
	URLSlug string
	// SiteMovies maps site ID -> the movie record (and its site-specific ID)
	// at that theater.
	SiteMovies map[string]Movie
}

// SiteIDs returns the theater IDs showing this title.
func (g TitleGroup) SiteIDs() []string {
	ids := make([]string, 0, len(g.SiteMovies))
	for id := range g.SiteMovies {
		ids = append(ids, id)
	}
	return ids
}

// GroupByTitle collapses a per-site movie list (e.g. from ListMovies over many
// sites) into one entry per film, keyed by URL slug, recording which theater
// shows it and under which site-specific ID. This answers "which cinemas are
// showing this movie?".
func GroupByTitle(movies []Movie) map[string]TitleGroup {
	groups := make(map[string]TitleGroup)
	for _, m := range movies {
		key := m.URLSlug
		if key == "" {
			key = m.Name
		}
		g, ok := groups[key]
		if !ok {
			g = TitleGroup{Name: m.Name, URLSlug: m.URLSlug, SiteMovies: map[string]Movie{}}
		}
		g.SiteMovies[m.SiteID] = m
		groups[key] = g
	}
	return groups
}
