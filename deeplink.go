package caribbeancinemas

import "net/url"

// DefaultSiteBaseURL is the Caribbean Cinemas consumer website.
const DefaultSiteBaseURL = "https://home.caribbeancinemas.com"

// Deeplinker builds links to the official Caribbean Cinemas website.
type Deeplinker struct {
	base string
}

// NewDeeplinker returns a Deeplinker. An empty baseURL uses DefaultSiteBaseURL.
func NewDeeplinker(baseURL string) *Deeplinker {
	if baseURL == "" {
		baseURL = DefaultSiteBaseURL
	}
	return &Deeplinker{base: baseURL}
}

// SeatSelectionURL links to seat selection for a showing at a theater.
func (d *Deeplinker) SeatSelectionURL(siteSlug, showingID string) string {
	return d.base + "/" + url.PathEscape(siteSlug) + "/checkout/seats/" + url.PathEscape(showingID)
}

// MovieURL links to a movie's page at a specific theater.
func (d *Deeplinker) MovieURL(siteSlug, movieSlug string) string {
	return d.base + "/" + url.PathEscape(siteSlug) + "/movie/" + url.PathEscape(movieSlug)
}
