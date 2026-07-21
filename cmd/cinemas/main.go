// Command cinemas is an unofficial terminal client for Caribbean Cinemas:
// browse movies, theaters, showtimes, seat maps, and prices, then open the
// official checkout.
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
		Short:   "Unofficial Caribbean Cinemas CLI",
		Version: buildVersion(),
		Long: "cinemas is an unofficial, community-built CLI for Caribbean Cinemas\n" +
			"(Puerto Rico). It is not affiliated with, maintained by, or endorsed by\n" +
			"Caribbean Cinemas. It provides read-only access to movies, theaters,\n" +
			"showtimes, seat availability, and prices. Ticket purchases open on the\n" +
			"official Caribbean Cinemas website in the default browser.",
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
