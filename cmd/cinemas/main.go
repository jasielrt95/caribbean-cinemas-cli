// Command cinemas is a terminal client for Caribbean Cinemas: browse movies,
// theaters, showtimes, seat maps, and prices, then open the official checkout.
package main

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:     "cinemas",
		Short:   "Caribbean Cinemas from your terminal",
		Version: buildVersion(),
		Long: "cinemas queries Caribbean Cinemas (Puerto Rico) for movies, theaters,\n" +
			"showtimes, seat availability, and prices. It is read-only; to buy\n" +
			"tickets it opens the official Caribbean Cinemas site in your browser.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(
		interactiveCmd(),
		theatersCmd(),
		moviesCmd(),
		showtimesCmd(),
		seatsCmd(),
		priceCmd(),
		checkoutCmd(),
		buyLinkCmd(),
	)
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func buildVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok || info.Main.Version == "" || info.Main.Version == "(devel)" {
		return "dev"
	}
	return info.Main.Version
}
