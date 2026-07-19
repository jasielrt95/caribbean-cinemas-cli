package caribbeancinemas

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"time"
)

const showtimesQuery = `query ($id: ID!) {
  movie(id: $id) {
    id
    showings {
      id time screenId ticketsSold displayMetaData
      screen { id name number seatCount }
      priceCard { id name }
    }
  }
}`

// Showtimes returns all scheduled showings for a movie, given its site-specific
// ID (from ListMovies/MoviesAtSite). The showings belong to the theater that ID
// is scoped to. Results are sorted by start time.
func (c *Client) Showtimes(ctx context.Context, movieID string) ([]Showing, error) {
	vars := map[string]any{"id": movieID}
	var resp struct {
		Movie *struct {
			Showings []Showing `json:"showings"`
		} `json:"movie"`
	}
	if err := c.execute(ctx, "", showtimesQuery, vars, &resp); err != nil {
		return nil, err
	}
	if resp.Movie == nil {
		return nil, ErrNotFound
	}
	showings := resp.Movie.Showings
	sort.Slice(showings, func(i, j int) bool { return showings[i].Time < showings[j].Time })
	return showings, nil
}

// StartTime parses the showing's UTC start time.
func (s Showing) StartTime() (time.Time, error) {
	return time.Parse(time.RFC3339, s.Time)
}

// IsUpcoming reports whether the showing starts at or after now. Showings whose
// time cannot be parsed are treated as not upcoming.
func (s Showing) IsUpcoming(now time.Time) bool {
	t, err := s.StartTime()
	if err != nil {
		return false
	}
	return !t.Before(now)
}

// FilterUpcoming returns only the showings that start at or after now, i.e. the
// ones still bookable. Input order is preserved.
func FilterUpcoming(showings []Showing, now time.Time) []Showing {
	out := make([]Showing, 0, len(showings))
	for _, s := range showings {
		if s.IsUpcoming(now) {
			out = append(out, s)
		}
	}
	return out
}

// UpcomingShowtimes is like Showtimes but returns only showings that have not
// yet started, using the current time.
func (c *Client) UpcomingShowtimes(ctx context.Context, movieID string) ([]Showing, error) {
	showings, err := c.Showtimes(ctx, movieID)
	if err != nil {
		return nil, err
	}
	return FilterUpcoming(showings, time.Now()), nil
}

// Format returns the presentation tags for this specific showing, parsed from
// the displayMetaData JSON (e.g. "spanish-dubbed", "spanish-subtitles",
// "cine-kids"). Returns nil if none are present.
func (s Showing) Format() []string {
	if s.DisplayMetaData == "" {
		return nil
	}
	var meta struct {
		Classes string `json:"classes"`
	}
	if err := json.Unmarshal([]byte(s.DisplayMetaData), &meta); err != nil {
		return nil
	}
	if strings.TrimSpace(meta.Classes) == "" {
		return nil
	}
	return strings.Fields(meta.Classes)
}

const showingMovieQuery = `query ($id: ID!) {
  showing(id: $id) { id movie { id name urlSlug siteId } }
}`

// MovieForShowing returns the movie a showing belongs to (name, slug, etc.),
// useful for building a purchase handoff link from a showing ID.
func (c *Client) MovieForShowing(ctx context.Context, showingID string) (*Movie, error) {
	vars := map[string]any{"id": showingID}
	var resp struct {
		Showing *struct {
			Movie *Movie `json:"movie"`
		} `json:"showing"`
	}
	if err := c.execute(ctx, "", showingMovieQuery, vars, &resp); err != nil {
		return nil, err
	}
	if resp.Showing == nil || resp.Showing.Movie == nil {
		return nil, ErrNotFound
	}
	return resp.Showing.Movie, nil
}

const pricingQuery = `query ($id: ID!) {
  showing(id: $id) {
    id
    priceCard { id name }
    tickets { price ticketType { id name displayOrder } }
  }
}`

// PriceSheet is the set of ticket prices for a showing.
type PriceSheet struct {
	ShowingID string
	PriceCard *PriceCard
	// Prices is one entry per ticket type (deduplicated), sorted by the
	// theater's display order.
	Prices []TicketType
}

// Pricing returns the ticket prices exposed by the public API for a showing.
// The API returns one ticket per seat, so this deduplicates by ticket type into
// a clean price sheet. An empty Prices slice means public pricing is
// unavailable; it does not necessarily mean the official checkout is closed.
func (c *Client) Pricing(ctx context.Context, showingID string) (*PriceSheet, error) {
	vars := map[string]any{"id": showingID}
	var resp struct {
		Showing *struct {
			ID        string     `json:"id"`
			PriceCard *PriceCard `json:"priceCard"`
			Tickets   []Ticket   `json:"tickets"`
		} `json:"showing"`
	}
	if err := c.execute(ctx, "", pricingQuery, vars, &resp); err != nil {
		return nil, err
	}
	if resp.Showing == nil {
		return nil, ErrNotFound
	}
	seen := map[string]TicketType{}
	for _, t := range resp.Showing.Tickets {
		if t.TicketType == nil {
			continue
		}
		tt := *t.TicketType
		tt.Price = t.Price // price lives on the ticket, not the type
		seen[tt.ID] = tt
	}
	prices := make([]TicketType, 0, len(seen))
	for _, tt := range seen {
		prices = append(prices, tt)
	}
	sort.Slice(prices, func(i, j int) bool {
		if prices[i].DisplayOrder != prices[j].DisplayOrder {
			return prices[i].DisplayOrder < prices[j].DisplayOrder
		}
		return prices[i].Price > prices[j].Price
	})
	return &PriceSheet{ShowingID: resp.Showing.ID, PriceCard: resp.Showing.PriceCard, Prices: prices}, nil
}
