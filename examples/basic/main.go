// Command basic demonstrates the browse -> showtimes -> seats -> handoff flow.
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	cc "github.com/jasielrt/caribbeancinemas-go"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := cc.New()
	const site = "45" // Plaza Las Américas

	movies, err := client.MoviesAtSite(ctx, site)
	if err != nil {
		log.Fatal(err)
	}
	if len(movies) == 0 {
		log.Fatal("no movies playing")
	}
	movie := movies[0]
	fmt.Printf("Now playing: %s (%s, %d min)\n", movie.Name, movie.Rating, movie.Duration)

	showings, err := client.Showtimes(ctx, movie.ID)
	if err != nil {
		log.Fatal(err)
	}
	if len(showings) == 0 {
		log.Fatal("no showtimes")
	}
	s := showings[0]
	when, _ := s.StartTime()
	fmt.Printf("Next showtime: %s (format: %v)\n", when.Local().Format(time.Kitchen), s.Format())

	if sheet, err := client.Pricing(ctx, s.ID); err == nil {
		for _, p := range sheet.Prices {
			fmt.Printf("  %-12s $%.2f\n", p.Name, p.Price)
		}
	}

	if chart, err := client.SeatChartForShowing(ctx, s.ID); err == nil {
		fmt.Printf("Seats: %d of %d available\n", len(chart.AvailableSeats()), chart.SeatCount)
	}

	link := ""
	if site, ok := cc.TheaterByID(movie.SiteID); ok {
		link = cc.NewDeeplinker("").SeatSelectionURL(site.Slug, s.ID)
	}
	fmt.Printf("Buy: %s\n", link)
}
