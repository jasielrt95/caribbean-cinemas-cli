package caribbeancinemas

import (
	"context"
	_ "embed"
	"encoding/json"
	"sort"
	"strings"
	"sync"
)

//go:embed sites.json
var sitesJSON []byte

//go:embed screens.json
var screensJSON []byte

// ScreenInfo maps a screen (auditorium) to its theater. Because the live
// `screens` query is scoped to one theater at a time via the site-id header,
// this circuit-wide mapping is captured here so a showing's ScreenID can be
// resolved to a theater without extra requests.
type ScreenInfo struct {
	SiteID    string `json:"siteId"`
	ScreenID  string `json:"screenId"`
	Name      string `json:"name"`
	Number    int    `json:"number"`
	SeatCount int    `json:"seatCount"`
}

var (
	embedOnce      sync.Once
	embeddedSites  []Site
	embeddedScreen map[string]ScreenInfo
)

func loadEmbedded() {
	embedOnce.Do(func() {
		_ = json.Unmarshal(sitesJSON, &embeddedSites)
		sort.Slice(embeddedSites, func(i, j int) bool {
			return embeddedSites[i].DisplayOrder < embeddedSites[j].DisplayOrder
		})
		var screens []ScreenInfo
		_ = json.Unmarshal(screensJSON, &screens)
		embeddedScreen = make(map[string]ScreenInfo, len(screens))
		for _, s := range screens {
			embeddedScreen[s.ScreenID] = s
		}
	})
}

// Theaters returns the built-in directory of all Caribbean Cinemas theaters,
// sorted by display order. This is embedded data and makes no network request.
func Theaters() []Site {
	loadEmbedded()
	out := make([]Site, len(embeddedSites))
	copy(out, embeddedSites)
	return out
}

// AllSites returns every theater's site ID, suitable for a circuit-wide movie
// listing. Embedded; no network request.
func AllSites() []string {
	loadEmbedded()
	ids := make([]string, len(embeddedSites))
	for i, s := range embeddedSites {
		ids[i] = s.ID
	}
	return ids
}

// TheaterByID returns the embedded theater record for a site ID, or false.
func TheaterByID(siteID string) (Site, bool) {
	loadEmbedded()
	for _, s := range embeddedSites {
		if s.ID == siteID {
			return s, true
		}
	}
	return Site{}, false
}

// ScreenTheater resolves a screen ID to its theater using embedded data,
// returning the ScreenInfo and whether it was found. A small number of
// historical screens may be absent.
func ScreenTheater(screenID string) (ScreenInfo, bool) {
	loadEmbedded()
	info, ok := embeddedScreen[screenID]
	return info, ok
}

const siteQuery = `query ($id: ID!) {
  site(id: $id) {
    id name abbreviation hostname address1 city state zip phone email lat lon timeZone
    facebook instagram twitter youtube tiktok displayOrder circuitId
  }
}`

// FetchSite fetches a theater's current record live from the API (rather than
// the embedded snapshot). Returns ErrNotFound if the ID is unknown.
func (c *Client) FetchSite(ctx context.Context, siteID string) (*Site, error) {
	vars := map[string]any{"id": siteID}
	var resp struct {
		Site *Site `json:"site"`
	}
	if err := c.execute(ctx, "", siteQuery, vars, &resp); err != nil {
		return nil, err
	}
	if resp.Site == nil {
		return nil, ErrNotFound
	}
	resp.Site.Slug = slugFromHostname(resp.Site.Hostname)
	return resp.Site, nil
}

func slugFromHostname(hostname string) string {
	h := strings.TrimRight(hostname, "/")
	if i := strings.LastIndex(h, "/"); i >= 0 {
		return h[i+1:]
	}
	return ""
}

const screensQuery = `query { screens { data { id name number seatCount } count } }`

// FetchScreens lists the auditoriums at one theater, live. This query is scoped
// by the site-id header, so it returns only the given theater's screens.
func (c *Client) FetchScreens(ctx context.Context, siteID string) ([]Screen, error) {
	var resp struct {
		Screens struct {
			Data []Screen `json:"data"`
		} `json:"screens"`
	}
	if err := c.execute(ctx, siteID, screensQuery, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Screens.Data, nil
}

const titleClassesQuery = `query { titleClasses { id name description displayOrder } }`

// FetchTitleClasses lists the content categories (Film, Ópera, Ballet, ...).
func (c *Client) FetchTitleClasses(ctx context.Context) ([]TitleClass, error) {
	var resp struct {
		TitleClasses []TitleClass `json:"titleClasses"`
	}
	if err := c.execute(ctx, "", titleClassesQuery, nil, &resp); err != nil {
		return nil, err
	}
	return resp.TitleClasses, nil
}
