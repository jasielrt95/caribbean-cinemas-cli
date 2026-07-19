package caribbeancinemas

import (
	"testing"
	"time"
)

func TestGroupByTitle(t *testing.T) {
	movies := []Movie{
		{ID: "1", Name: "Moana", URLSlug: "moana", SiteID: "45"},
		{ID: "2", Name: "Moana", URLSlug: "moana", SiteID: "2"},
		{ID: "3", Name: "Evil Dead", URLSlug: "evil-dead", SiteID: "45"},
	}
	groups := GroupByTitle(movies)
	if len(groups) != 2 {
		t.Fatalf("want 2 groups, got %d", len(groups))
	}
	moana := groups["moana"]
	if len(moana.SiteMovies) != 2 {
		t.Errorf("Moana should play at 2 sites, got %d", len(moana.SiteMovies))
	}
	if moana.SiteMovies["45"].ID != "1" {
		t.Errorf("site 45 Moana ID = %q, want 1", moana.SiteMovies["45"].ID)
	}
}

func TestShowingFormat(t *testing.T) {
	tests := []struct {
		name string
		meta string
		want int
	}{
		{"single", `{"classes":"spanish-dubbed"}`, 1},
		{"multiple", `{"classes":"spanish-subtitles cxc"}`, 2},
		{"empty classes", `{"classes":""}`, 0},
		{"no metadata", ``, 0},
		{"malformed", `not json`, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Showing{DisplayMetaData: tt.meta}
			if got := len(s.Format()); got != tt.want {
				t.Errorf("Format() len = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestDeeplinker(t *testing.T) {
	d := NewDeeplinker("")
	if got, want := d.SeatSelectionURL("plaza-del-sol", "889676"), "https://home.caribbeancinemas.com/plaza-del-sol/checkout/seats/889676"; got != want {
		t.Errorf("SeatSelectionURL = %q, want %q", got, want)
	}
	if got, want := d.MovieURL("plaza-del-sol", "the-invite"), "https://home.caribbeancinemas.com/plaza-del-sol/movie/the-invite"; got != want {
		t.Errorf("MovieURL = %q, want %q", got, want)
	}
}

func TestEmbeddedTheaterSlugs(t *testing.T) {
	want := map[string]string{"52": "belz", "45": "plaza-americas", "2": "arecibo"}
	for id, slug := range want {
		s, ok := TheaterByID(id)
		if !ok {
			t.Fatalf("site %s missing", id)
		}
		if s.Slug != slug {
			t.Errorf("site %s slug = %q, want %q", id, s.Slug, slug)
		}
	}
	for _, s := range Theaters() {
		if s.Slug == "" {
			t.Errorf("theater %s (%s) has empty slug", s.ID, s.Name)
		}
	}
}

func TestFilterUpcoming(t *testing.T) {
	now := time.Date(2026, 7, 12, 18, 0, 0, 0, time.UTC)
	showings := []Showing{
		{ID: "past", Time: "2026-07-12T15:00:00Z"},
		{ID: "future", Time: "2026-07-12T20:00:00Z"},
		{ID: "now", Time: "2026-07-12T18:00:00Z"},
		{ID: "bad", Time: "not-a-time"},
	}
	up := FilterUpcoming(showings, now)
	if len(up) != 2 {
		t.Fatalf("want 2 upcoming, got %d", len(up))
	}
	if up[0].ID != "future" || up[1].ID != "now" {
		t.Errorf("unexpected upcoming set: %v, %v", up[0].ID, up[1].ID)
	}
}

func TestEmbeddedTheaters(t *testing.T) {
	theaters := Theaters()
	if len(theaters) != 31 {
		t.Fatalf("want 31 theaters, got %d", len(theaters))
	}
	for _, s := range theaters {
		if s.Name == "" || s.Address1 == "" {
			t.Errorf("theater %s missing name/address", s.ID)
		}
	}
	if s, ok := TheaterByID("45"); !ok || s.Name != "Plaza Las Americas" {
		t.Errorf("site 45 = %+v, ok=%v", s, ok)
	}
	if len(AllSites()) != 31 {
		t.Errorf("AllSites len = %d, want 31", len(AllSites()))
	}
}

func TestScreenTheater(t *testing.T) {
	if _, ok := ScreenTheater("754784"); !ok {
		t.Skip("sample screen not present in embedded data")
	}
}

func TestPosterURL(t *testing.T) {
	m := Movie{PosterImage: "abc123"}
	if got := m.PosterURL(400); got == "" {
		t.Error("PosterURL should be non-empty when PosterImage is set")
	}
	empty := Movie{}
	if got := empty.PosterURL(400); got != "" {
		t.Errorf("PosterURL = %q, want empty", got)
	}
}

func TestIsAuthRequired(t *testing.T) {
	if !IsAuthRequired(&APIError{Code: CodeUnauthenticated}) {
		t.Error("101 should be auth-required")
	}
	if IsAuthRequired(&APIError{Code: CodeValidation}) {
		t.Error("210 should not be auth-required")
	}
	if IsAuthRequired(ErrNotFound) {
		t.Error("ErrNotFound should not be auth-required")
	}
}
