package caribbeancinemas

// Movie is a film (or event) as it appears at a specific theater. Note the
// per-site ID model: the same title has a different ID at each theater, and
// SiteID indicates which theater this record belongs to. Not every field is
// populated for every title.
type Movie struct {
	ID                     string   `json:"id"`
	Name                   string   `json:"name"`
	Abbreviation           string   `json:"abbreviation"`
	URLSlug                string   `json:"urlSlug"`
	ShowingStatus          string   `json:"showingStatus"`
	Synopsis               string   `json:"synopsis"`
	Starring               string   `json:"starring"`
	DirectedBy             string   `json:"directedBy"`
	ProducedBy             string   `json:"producedBy"`
	Writers                string   `json:"writers"`
	Duration               int      `json:"duration"` // minutes
	Genre                  string   `json:"genre"`
	AllGenres              string   `json:"allGenres"`
	Rating                 string   `json:"rating"`
	RatingReason           string   `json:"ratingReason"`
	OriginalLanguage       string   `json:"originalLanguage"`
	CountryOfOrigin        string   `json:"countryOfOrigin"`
	TrailerYouTubeID       string   `json:"trailerYoutubeId"`
	PosterImage            string   `json:"posterImage"`
	BannerImage            string   `json:"bannerImage"`
	Color                  string   `json:"color"`
	ReleaseDate            string   `json:"releaseDate"`
	DateOfFirstShowing     string   `json:"dateOfFirstShowing"`
	DatesWithPublicShowing []string `json:"datesWithPublicShowing"`
	ShowingCount           int      `json:"showingCount"`
	IsMarathon             bool     `json:"isMarathon"`
	SiteID                 string   `json:"siteId"`
	TitleClassID           string   `json:"titleClassId"`
}

// PosterURL returns an imgix URL for the movie poster at the given width, or ""
// if the movie has no poster.
func (m Movie) PosterURL(width int) string {
	if m.PosterImage == "" {
		return ""
	}
	return imgixURL(m.PosterImage, width)
}

// Showing is a single scheduled screening at a theater. Times are UTC.
type Showing struct {
	ID              string     `json:"id"`
	Time            string     `json:"time"` // RFC3339 UTC, e.g. 2026-07-10T17:20:00Z
	ScreenID        string     `json:"screenId"`
	TicketsSold     int        `json:"ticketsSold"`
	DisplayMetaData string     `json:"displayMetaData"` // JSON string; see Format
	Screen          *Screen    `json:"screen"`
	PriceCard       *PriceCard `json:"priceCard"`
}

// Screen is an auditorium.
type Screen struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Number    int    `json:"number"`
	SeatCount int    `json:"seatCount"`
}

// PriceCard is the named pricing scheme applied to a showing (e.g. "Regular",
// "Anime"). It carries no price detail itself; use Client.Pricing.
type PriceCard struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// TicketType is a named ticket category with a price (e.g. Adult, Children).
type TicketType struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Price        float64 `json:"price"`
	DisplayOrder int     `json:"displayOrder"`
}

// Ticket is one priced ticket for a showing, linked to its type. The raw API
// returns one Ticket per seat; Client.Pricing deduplicates them into a price
// sheet.
type Ticket struct {
	Price      float64     `json:"price"`
	TicketType *TicketType `json:"ticketType"`
}

// Site is a theater. Address and geo detail vary: every theater has Address1,
// but Lat/Lon are only populated for some.
type Site struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Abbreviation string `json:"abbreviation"`
	// Slug is the theater's URL slug (from Hostname), e.g. "plaza-del-sol". It
	// is required to build seat-selection deep links and is NOT derivable from
	// the name (e.g. "The Outlet 66" -> "belz").
	Slug         string   `json:"slug"`
	Hostname     string   `json:"hostname"`
	Address1     string   `json:"address1"`
	City         string   `json:"city"`
	State        string   `json:"state"`
	Zip          string   `json:"zip"`
	Phone        string   `json:"phone"`
	Email        string   `json:"email"`
	Lat          *float64 `json:"lat"`
	Lon          *float64 `json:"lon"`
	TimeZone     string   `json:"timeZone"`
	Facebook     string   `json:"facebook"`
	Instagram    string   `json:"instagram"`
	Twitter      string   `json:"twitter"`
	YouTube      string   `json:"youtube"`
	TikTok       string   `json:"tiktok"`
	DisplayOrder int      `json:"displayOrder"`
	CircuitID    string   `json:"circuitId"`
}

// HasGeo reports whether the theater has usable latitude/longitude coordinates.
func (s Site) HasGeo() bool { return s.Lat != nil && s.Lon != nil }

// TitleClass is a content category (Film, Special Events, Ópera, Ballet, ...).
type TitleClass struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	DisplayOrder int    `json:"displayOrder"`
}
