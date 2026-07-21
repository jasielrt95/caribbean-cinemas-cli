package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	cc "github.com/jasielrt95/caribbean-cinemas-cli"
	"github.com/spf13/cobra"
)

func newClient() (*cc.Client, context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	return cc.New(), ctx, cancel
}

func tw() *tabwriter.Writer {
	return tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
}

func theatersCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "theaters",
		Short: "List all Caribbean Cinemas theaters",
		RunE: func(cmd *cobra.Command, args []string) error {
			w := tw()
			fmt.Fprintln(w, "ID\tTHEATER\tCITY\tADDRESS")
			for _, s := range cc.Theaters() {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", s.ID, s.Name, s.City, s.Address1)
			}
			return w.Flush()
		},
	}
}

func moviesCmd() *cobra.Command {
	var siteID string
	var comingSoon, future bool
	cmd := &cobra.Command{
		Use:   "movies",
		Short: "List now-playing movies (optionally at one theater)",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, ctx, cancel := newClient()
			defer cancel()

			opts := cc.MovieListOptions{}
			switch {
			case comingSoon:
				opts.Type = cc.TypeComingSoon
			case future:
				opts.Type = cc.TypeFuture
			}
			// Single theater: one call. Circuit-wide: fan out so each film's
			// full theater list is captured (a plain multi-site query collapses
			// each title to one representative theater).
			var movies []cc.Movie
			var err error
			if siteID != "" {
				opts.SiteIDs = []string{siteID}
				movies, err = client.ListMovies(ctx, opts)
			} else if opts.Type == "" || opts.Type == cc.TypeNowPlaying {
				movies, err = client.NowPlayingEverywhere(ctx, cc.AllSites())
			} else {
				opts.SiteIDs = cc.AllSites()
				movies, err = client.ListMovies(ctx, opts)
			}
			if err != nil {
				return err
			}

			if siteID != "" {
				w := tw()
				fmt.Fprintln(w, "MOVIE-ID\tNAME\tRATING\tGENRE\tMIN")
				for _, m := range movies {
					fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\n", m.ID, m.Name, m.Rating, m.Genre, m.Duration)
				}
				return w.Flush()
			}
			groups := cc.GroupByTitle(movies)
			w := tw()
			fmt.Fprintln(w, "SLUG\tNAME\tRATING\t#THEATERS")
			for _, g := range groups {
				var rating string
				for _, m := range g.SiteMovies {
					rating = m.Rating
					break
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%d\n", g.URLSlug, g.Name, rating, len(g.SiteMovies))
			}
			return w.Flush()
		},
	}
	cmd.Flags().StringVar(&siteID, "site", "", "limit to one theater (site ID); shows site-specific movie IDs")
	cmd.Flags().BoolVar(&comingSoon, "coming-soon", false, "list coming-soon instead of now-playing")
	cmd.Flags().BoolVar(&future, "future", false, "list future/event titles")
	return cmd
}

func showtimesCmd() *cobra.Command {
	var movieID, slug, siteID string
	var all bool
	cmd := &cobra.Command{
		Use:   "showtimes",
		Short: "Show upcoming showtimes for a movie at a theater",
		Long: "Provide either --movie-id (a site-specific movie ID from `cinemas movies --site`)\n" +
			"or --title <slug> together with --site to resolve it automatically.",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, ctx, cancel := newClient()
			defer cancel()

			id := movieID
			if id == "" {
				if slug == "" || siteID == "" {
					return fmt.Errorf("provide --movie-id, or both --title and --site")
				}
				movies, err := client.MoviesAtSite(ctx, siteID)
				if err != nil {
					return err
				}
				for _, m := range movies {
					if m.URLSlug == slug || strings.EqualFold(m.Name, slug) {
						id = m.ID
						break
					}
				}
				if id == "" {
					return fmt.Errorf("movie %q not found at site %s", slug, siteID)
				}
			}

			showings, err := client.Showtimes(ctx, id)
			if err != nil {
				return err
			}
			if !all {
				showings = cc.FilterUpcoming(showings, time.Now())
			}
			w := tw()
			fmt.Fprintln(w, "SHOWING-ID\tLOCAL TIME\tSCREEN\tFORMAT")
			for _, s := range showings {
				local := s.Time
				if t, err := s.StartTime(); err == nil {
					local = t.Local().Format("Mon Jan 2 3:04 PM")
				}
				screen := ""
				if s.Screen != nil {
					screen = s.Screen.Name
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", s.ID, local, screen, strings.Join(s.Format(), ","))
			}
			return w.Flush()
		},
	}
	cmd.Flags().StringVar(&movieID, "movie-id", "", "site-specific movie ID")
	cmd.Flags().StringVar(&slug, "title", "", "movie slug or name (with --site)")
	cmd.Flags().StringVar(&siteID, "site", "", "theater site ID (with --title)")
	cmd.Flags().BoolVar(&all, "all", false, "include past showtimes (default: upcoming only)")
	return cmd
}

func seatsCmd() *cobra.Command {
	var showingID string
	cmd := &cobra.Command{
		Use:   "seats",
		Short: "Show seat availability for a showing",
		RunE: func(cmd *cobra.Command, args []string) error {
			if showingID == "" {
				return fmt.Errorf("--showing is required")
			}
			client, ctx, cancel := newClient()
			defer cancel()

			chart, err := client.SeatChartForShowing(ctx, showingID)
			if err != nil {
				return err
			}
			avail := chart.AvailableSeats()
			fmt.Fprintf(os.Stdout, "%s — %d seats, %d available\n\n",
				chart.Name, chart.SeatCount, len(avail))
			for _, row := range chart.Grid {
				var b strings.Builder
				for _, seat := range row {
					switch {
					case seat.IsAisle():
						b.WriteByte(' ')
					case seat.Available:
						b.WriteByte('.')
					default:
						b.WriteByte('#')
					}
				}
				fmt.Fprintln(os.Stdout, b.String())
			}
			fmt.Fprintln(os.Stdout, "\nlegend: . open   # taken   (blank) aisle")
			return nil
		},
	}
	cmd.Flags().StringVar(&showingID, "showing", "", "showing ID (required)")
	return cmd
}

func priceCmd() *cobra.Command {
	var showingID string
	cmd := &cobra.Command{
		Use:   "price",
		Short: "Show ticket prices for a showing",
		RunE: func(cmd *cobra.Command, args []string) error {
			if showingID == "" {
				return fmt.Errorf("--showing is required")
			}
			client, ctx, cancel := newClient()
			defer cancel()

			sheet, err := client.Pricing(ctx, showingID)
			if err != nil {
				return err
			}
			card := "—"
			if sheet.PriceCard != nil {
				card = sheet.PriceCard.Name
			}
			fmt.Fprintf(os.Stdout, "Price card: %s\n", card)
			if len(sheet.Prices) == 0 {
				fmt.Fprintln(os.Stdout, "(public pricing unavailable; the official checkout may still offer tickets)")
				return nil
			}
			w := tw()
			fmt.Fprintln(w, "TICKET\tPRICE")
			for _, p := range sheet.Prices {
				fmt.Fprintf(w, "%s\t$%.2f\n", p.Name, p.Price)
			}
			return w.Flush()
		},
	}
	cmd.Flags().StringVar(&showingID, "showing", "", "showing ID (required)")
	return cmd
}

func buyLinkCmd() *cobra.Command {
	var showingID string
	cmd := &cobra.Command{
		Use:   "buy-link",
		Short: "Print the direct seat-selection link for a showing",
		Long: "Prints the official seat-selection URL for a showing. Opened in a\n" +
			"browser, it lands on the seat map for that exact showtime, where the\n" +
			"user can continue as guest, pick seats, and pay — no login required.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if showingID == "" {
				return fmt.Errorf("--showing is required")
			}
			client, ctx, cancel := newClient()
			defer cancel()

			url, err := officialCheckoutURL(ctx, client, showingID)
			if err != nil {
				return err
			}
			fmt.Fprintln(os.Stdout, url)
			return nil
		},
	}
	cmd.Flags().StringVar(&showingID, "showing", "", "showing ID (required)")
	return cmd
}

func checkoutCmd() *cobra.Command {
	var showingID string
	cmd := &cobra.Command{
		Use:   "checkout",
		Short: "Open official seat selection in your browser",
		Long: "Opens the official Caribbean Cinemas seat-selection page for a showing.\n" +
			"The CLI does not select seats, create an order, or handle payment.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if showingID == "" {
				return fmt.Errorf("--showing is required")
			}
			client, ctx, cancel := newClient()
			defer cancel()

			url, err := officialCheckoutURL(ctx, client, showingID)
			if err != nil {
				return err
			}
			if err := openBrowser(url); err != nil {
				return fmt.Errorf("open official checkout: %w", err)
			}
			fmt.Fprintln(os.Stdout, "Opened the official Caribbean Cinemas checkout in your browser.")
			return nil
		},
	}
	cmd.Flags().StringVar(&showingID, "showing", "", "showing ID (required)")
	return cmd
}

func officialCheckoutURL(ctx context.Context, client *cc.Client, showingID string) (string, error) {
	movie, err := client.MovieForShowing(ctx, showingID)
	if err != nil {
		return "", err
	}
	site, ok := cc.TheaterByID(movie.SiteID)
	if !ok || site.Slug == "" {
		return "", fmt.Errorf("unknown theater for showing (site %s)", movie.SiteID)
	}
	return cc.NewDeeplinker("").SeatSelectionURL(site.Slug, showingID), nil
}
