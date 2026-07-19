package caribbeancinemas

import (
	"context"
	"encoding/json"
	"fmt"
)

// Seat availability values (Seat.AvailabilityType).
const (
	SeatAvailable   = "available"
	SeatUnavailable = "unavailable"
	SeatBroken      = "broken"
	SeatHouse       = "house"
	SeatReserved    = "reserved"
	SeatAisle       = "aisle"
)

// Seat is a single position in an auditorium. A seat with SeatType "aisle" is a
// gap, not a real seat.
type Seat struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	SeatType         string `json:"seatType"`
	Available        bool   `json:"available"`
	AvailabilityType string `json:"availabilityType"`
}

// IsAisle reports whether this cell is an aisle/gap rather than a seat.
func (s Seat) IsAisle() bool { return s.SeatType == SeatAisle || s.AvailabilityType == SeatAisle }

// SeatChartDefinition describes the auditorium grid.
type SeatChartDefinition struct {
	RowCount        int      `json:"rowCount"`
	ColumnCount     int      `json:"columnCount"`
	PrimarySeatType string   `json:"primarySeatType"`
	RowAisles       []string `json:"rowAisles"`
	ColumnAisles    []int    `json:"columnAisles"`
}

// SeatChartOptions holds layout/sale options for the chart.
type SeatChartOptions struct {
	SocialDistancing      bool `json:"socialDistancing"`
	ColumnGap             int  `json:"columnGap"`
	RowGap                int  `json:"rowGap"`
	AllowGaps             bool `json:"allowGaps"`
	AllowCorners          bool `json:"allowCorners"`
	AllowSeatOrphans      bool `json:"allowSeatOrphans"`
	AllowPartialCouchSale bool `json:"allowPartialCouchSale"`
}

// SeatChart is the parsed seat map for a showing. Grid is indexed [row][column].
type SeatChart struct {
	ID         string
	Name       string
	SeatCount  int
	Grid       [][]Seat
	Definition SeatChartDefinition
	Options    SeatChartOptions
}

// AvailableSeats returns all real (non-aisle) seats that are currently open.
func (sc SeatChart) AvailableSeats() []Seat {
	var out []Seat
	for _, row := range sc.Grid {
		for _, seat := range row {
			if !seat.IsAisle() && seat.Available {
				out = append(out, seat)
			}
		}
	}
	return out
}

// rawSeatChart mirrors the GraphQL response, where the three payloads arrive as
// JSON-encoded strings that must be parsed a second time.
type rawSeatChart struct {
	ID                  string `json:"id"`
	Name                string `json:"name"`
	SeatCount           int    `json:"seatCount"`
	SeatChart           string `json:"seatChart"`
	SeatChartDefinition string `json:"seatChartDefinition"`
	SeatChartOptions    string `json:"seatChartOptions"`
}

const seatChartQuery = `query ($showingId: ID!) {
  seatChartForShowing(showingId: $showingId) {
    id name seatCount seatChart seatChartDefinition seatChartOptions
  }
}`

// SeatChartForShowing fetches and parses the seat map for a showing. The
// availability it reports is a point-in-time snapshot, not a live-locked view.
func (c *Client) SeatChartForShowing(ctx context.Context, showingID string) (*SeatChart, error) {
	vars := map[string]any{"showingId": showingID}
	var resp struct {
		SeatChartForShowing *rawSeatChart `json:"seatChartForShowing"`
	}
	if err := c.execute(ctx, "", seatChartQuery, vars, &resp); err != nil {
		return nil, err
	}
	r := resp.SeatChartForShowing
	if r == nil {
		return nil, ErrNotFound
	}

	sc := &SeatChart{ID: r.ID, Name: r.Name, SeatCount: r.SeatCount}
	if r.SeatChart != "" {
		if err := json.Unmarshal([]byte(r.SeatChart), &sc.Grid); err != nil {
			return nil, fmt.Errorf("caribbeancinemas: parse seatChart: %w", err)
		}
	}
	if r.SeatChartDefinition != "" {
		if err := json.Unmarshal([]byte(r.SeatChartDefinition), &sc.Definition); err != nil {
			return nil, fmt.Errorf("caribbeancinemas: parse seatChartDefinition: %w", err)
		}
	}
	if r.SeatChartOptions != "" {
		if err := json.Unmarshal([]byte(r.SeatChartOptions), &sc.Options); err != nil {
			return nil, fmt.Errorf("caribbeancinemas: parse seatChartOptions: %w", err)
		}
	}
	return sc, nil
}
