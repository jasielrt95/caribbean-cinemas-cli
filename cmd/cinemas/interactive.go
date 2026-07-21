package main

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	cc "github.com/jasielrt95/caribbean-cinemas-cli"
	"github.com/spf13/cobra"
)

func interactiveCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "interactive",
		Aliases: []string{"browse", "i", "tui"},
		Short:   "Guided movie, theater, showtime, seats, and browser handoff",
		RunE: func(cmd *cobra.Command, args []string) error {
			p := tea.NewProgram(newTUI(), tea.WithAltScreen())
			_, err := p.Run()
			return err
		},
	}
}

type screen int

const (
	scrMovies screen = iota
	scrTheaters
	scrShowtimes
	scrDetail
)

var (
	stTitle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")).Background(lipgloss.Color("63")).Padding(0, 1)
	stCrumb   = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	stHelp    = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	stErr     = lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Bold(true)
	stLabel   = lipgloss.NewStyle().Foreground(lipgloss.Color("111")).Bold(true)
	stPrice   = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	stURL     = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Underline(true)
	seatOpen  = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	seatTaken = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	seatScrn  = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
)

type movieItem struct{ g cc.TitleGroup }

func (i movieItem) Title() string { return i.g.Name }
func (i movieItem) Description() string {
	n := len(i.g.SiteMovies)
	return fmt.Sprintf("%d theater%s", n, plural(n))
}
func (i movieItem) FilterValue() string { return i.g.Name }

type theaterItem struct {
	site    cc.Site
	movieID string
}

func (i theaterItem) Title() string       { return i.site.Name }
func (i theaterItem) Description() string { return i.site.City }
func (i theaterItem) FilterValue() string { return i.site.Name + " " + i.site.City }

type showtimeItem struct{ s cc.Showing }

func (i showtimeItem) Title() string {
	when := i.s.Time
	if t, err := i.s.StartTime(); err == nil {
		when = t.Local().Format("Mon Jan 2, 3:04 PM")
	}
	return when
}
func (i showtimeItem) Description() string {
	parts := []string{}
	if i.s.Screen != nil {
		parts = append(parts, i.s.Screen.Name)
	}
	if f := i.s.Format(); len(f) > 0 {
		parts = append(parts, strings.Join(f, ", "))
	}
	return strings.Join(parts, " · ")
}
func (i showtimeItem) FilterValue() string { return i.Title() }

type detailData struct {
	showing   cc.Showing
	priceCard string
	prices    []cc.TicketType
	chart     *cc.SeatChart
	buyURL    string
}

type tuiModel struct {
	client  *cc.Client
	screen  screen
	loading bool
	spinner spinner.Model
	status  string
	err     error
	w, h    int

	movies    list.Model
	theaters  list.Model
	showtimes list.Model
	// A zero-value list.Model panics on SetSize.
	moviesInit    bool
	theatersInit  bool
	showtimesInit bool

	curMovie cc.TitleGroup
	curSite  cc.Site
	detail   *detailData
}

func newTUI() tuiModel {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
	return tuiModel{client: cc.New(), spinner: sp, loading: true, status: "Loading what's playing across all theaters…"}
}

func (m tuiModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, loadMoviesCmd(m.client))
}

type moviesMsg struct {
	movies []cc.Movie
	err    error
}
type showtimesMsg struct {
	showings []cc.Showing
	err      error
}
type detailMsg struct {
	d   *detailData
	err error
}
type browserOpenedMsg struct{ err error }

func loadMoviesCmd(c *cc.Client) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		m, err := c.NowPlayingEverywhere(ctx, cc.AllSites())
		return moviesMsg{m, err}
	}
}

func loadShowtimesCmd(c *cc.Client, movieID string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		s, err := c.UpcomingShowtimes(ctx, movieID)
		return showtimesMsg{s, err}
	}
}

func loadDetailCmd(c *cc.Client, s cc.Showing, siteSlug string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		d := &detailData{showing: s, buyURL: cc.NewDeeplinker("").SeatSelectionURL(siteSlug, s.ID)}
		if sheet, err := c.Pricing(ctx, s.ID); err == nil {
			if sheet.PriceCard != nil {
				d.priceCard = sheet.PriceCard.Name
			}
			d.prices = sheet.Prices
		}
		chart, err := c.SeatChartForShowing(ctx, s.ID)
		if err != nil {
			return detailMsg{nil, err}
		}
		d.chart = chart
		return detailMsg{d, nil}
	}
}

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		lw, lh := msg.Width, msg.Height-4
		if m.moviesInit {
			m.movies.SetSize(lw, lh)
		}
		if m.theatersInit {
			m.theaters.SetSize(lw, lh)
		}
		if m.showtimesInit {
			m.showtimes.SetSize(lw, lh)
		}
		return m, nil

	case moviesMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		items := []list.Item{}
		groups := cc.GroupByTitle(msg.movies)
		titles := make([]cc.TitleGroup, 0, len(groups))
		for _, g := range groups {
			titles = append(titles, g)
		}
		sort.Slice(titles, func(i, j int) bool { return titles[i].Name < titles[j].Name })
		for _, g := range titles {
			items = append(items, movieItem{g})
		}
		m.movies = newList(items, "Now Playing — pick a movie", m.w, m.h)
		m.moviesInit = true
		m.screen = scrMovies
		return m, nil

	case showtimesMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		items := make([]list.Item, 0, len(msg.showings))
		for _, s := range msg.showings {
			items = append(items, showtimeItem{s})
		}
		title := fmt.Sprintf("%s @ %s — upcoming showtimes", m.curMovie.Name, m.curSite.Name)
		m.showtimes = newList(items, title, m.w, m.h)
		m.showtimesInit = true
		m.screen = scrShowtimes
		return m, nil

	case detailMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.detail = msg.d
		m.screen = scrDetail
		return m, nil

	case browserOpenedMsg:
		if msg.err != nil {
			m.err = fmt.Errorf("open official checkout: %w", msg.err)
			return m, nil
		}
		m.status = "Opened the official Caribbean Cinemas checkout in the default browser."
		return m, nil

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m.updateActiveList(msg)
}

func (m tuiModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	filtering := m.activeFiltering()

	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "q":
		if !filtering {
			return m, tea.Quit
		}
	case "esc", "b":
		if !filtering {
			return m.goBack()
		}
	case "enter":
		if !filtering {
			return m.advance()
		}
	case "o", "O":
		if !filtering && m.screen == scrDetail && m.detail != nil {
			return m, openBrowserCmd(m.detail.buyURL)
		}
	}
	return m.updateActiveList(msg)
}

func (m tuiModel) advance() (tea.Model, tea.Cmd) {
	switch m.screen {
	case scrMovies:
		it, ok := m.movies.SelectedItem().(movieItem)
		if !ok {
			return m, nil
		}
		m.curMovie = it.g
		var theaters []theaterItem
		for siteID, mv := range it.g.SiteMovies {
			site, ok := cc.TheaterByID(siteID)
			if !ok {
				site = cc.Site{ID: siteID, Name: "Site " + siteID}
			}
			theaters = append(theaters, theaterItem{site: site, movieID: mv.ID})
		}
		sort.Slice(theaters, func(i, j int) bool { return theaters[i].site.DisplayOrder < theaters[j].site.DisplayOrder })
		items := make([]list.Item, len(theaters))
		for i, t := range theaters {
			items[i] = t
		}
		m.theaters = newList(items, it.g.Name+" — pick a theater", m.w, m.h)
		m.theatersInit = true
		m.screen = scrTheaters
		return m, nil

	case scrTheaters:
		it, ok := m.theaters.SelectedItem().(theaterItem)
		if !ok {
			return m, nil
		}
		m.curSite = it.site
		m.loading = true
		m.status = "Loading showtimes…"
		return m, tea.Batch(m.spinner.Tick, loadShowtimesCmd(m.client, it.movieID))

	case scrShowtimes:
		it, ok := m.showtimes.SelectedItem().(showtimeItem)
		if !ok {
			return m, nil
		}
		m.loading = true
		m.status = "Loading prices & seats…"
		return m, tea.Batch(m.spinner.Tick, loadDetailCmd(m.client, it.s, m.curSite.Slug))
	}
	return m, nil
}

func (m tuiModel) goBack() (tea.Model, tea.Cmd) {
	switch m.screen {
	case scrTheaters:
		m.screen = scrMovies
	case scrShowtimes:
		m.screen = scrTheaters
	case scrDetail:
		m.screen = scrShowtimes
	case scrMovies:
		return m, tea.Quit
	}
	m.err = nil
	return m, nil
}

func (m *tuiModel) activeList() *list.Model {
	switch m.screen {
	case scrMovies:
		return &m.movies
	case scrTheaters:
		return &m.theaters
	case scrShowtimes:
		return &m.showtimes
	}
	return nil
}

func (m tuiModel) activeFiltering() bool {
	l := (&m).activeList()
	return l != nil && l.FilterState() == list.Filtering
}

func (m tuiModel) updateActiveList(msg tea.Msg) (tea.Model, tea.Cmd) {
	l := (&m).activeList()
	if l == nil {
		return m, nil
	}
	var cmd tea.Cmd
	*l, cmd = l.Update(msg)
	return m, cmd
}

func (m tuiModel) View() string {
	if m.err != nil {
		return "\n" + stErr.Render("Error: "+m.err.Error()) + "\n\n" + stHelp.Render("b back · q quit") + "\n"
	}
	if m.loading {
		return fmt.Sprintf("\n  %s %s\n", m.spinner.View(), m.status)
	}
	switch m.screen {
	case scrMovies:
		return m.movies.View()
	case scrTheaters:
		return m.theaters.View()
	case scrShowtimes:
		return m.showtimes.View()
	case scrDetail:
		return m.detailView()
	}
	return ""
}

func (m tuiModel) detailView() string {
	d := m.detail
	var b strings.Builder

	when := d.showing.Time
	if t, err := d.showing.StartTime(); err == nil {
		when = t.Local().Format("Mon Jan 2, 3:04 PM")
	}
	b.WriteString(stTitle.Render(m.curMovie.Name) + "  " + stCrumb.Render(m.curSite.Name+" · "+when))
	b.WriteString("\n\n")

	card := d.priceCard
	if card == "" {
		card = "—"
	}
	b.WriteString(stLabel.Render("Prices ") + stCrumb.Render("("+card+")") + "\n")
	for _, p := range d.prices {
		b.WriteString(fmt.Sprintf("  %-16s %s\n", p.Name, stPrice.Render(fmt.Sprintf("$%.2f", p.Price))))
	}
	if len(d.prices) == 0 {
		b.WriteString(stHelp.Render("  (public pricing unavailable; see the official checkout)\n"))
	}

	if d.chart != nil {
		b.WriteString("\n" + stLabel.Render("Seats") +
			stCrumb.Render(fmt.Sprintf(" — %s · %d of %d open", d.chart.Name, len(d.chart.AvailableSeats()), d.chart.SeatCount)) + "\n")
		b.WriteString(seatScrn.Render("  "+centeredScreenBar(d.chart)) + "\n")
		b.WriteString(renderSeatMap(d.chart))
		b.WriteString(stHelp.Render("  "+seatOpen.Render("•")+" open   "+seatTaken.Render("•")+" taken") + "\n")
	}

	b.WriteString("\n" + stLabel.Render("Buy tickets") + stCrumb.Render(" — opens seat selection (continue as guest)") + "\n  " + stURL.Render(d.buyURL) + "\n")
	if strings.HasPrefix(m.status, "Opened the official") {
		b.WriteString("\n" + stPrice.Render(m.status) + "\n")
	}
	b.WriteString("\n" + stHelp.Render("o open in browser · b back · q quit") + "\n")
	return b.String()
}

func renderSeatMap(chart *cc.SeatChart) string {
	var b strings.Builder
	for _, row := range chart.Grid {
		b.WriteString("  ")
		for _, seat := range row {
			switch {
			case seat.IsAisle():
				b.WriteByte(' ')
			case seat.Available:
				b.WriteString(seatOpen.Render("•"))
			default:
				b.WriteString(seatTaken.Render("•"))
			}
		}
		b.WriteByte('\n')
	}
	return b.String()
}

func centeredScreenBar(chart *cc.SeatChart) string {
	width := chart.Definition.ColumnCount
	if width <= 0 && len(chart.Grid) > 0 {
		width = len(chart.Grid[0])
	}
	if width < 6 {
		width = 6
	}
	label := " SCREEN "
	if len(label) >= width {
		return label
	}
	pad := (width - len(label)) / 2
	return strings.Repeat("▁", pad) + label + strings.Repeat("▁", width-len(label)-pad)
}

func newList(items []list.Item, title string, w, h int) list.Model {
	if w == 0 {
		w = 80
	}
	if h == 0 {
		h = 24
	}
	l := list.New(items, list.NewDefaultDelegate(), w, h-4)
	l.Title = title
	l.Styles.Title = stTitle
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(true)
	return l
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func openBrowser(url string) error {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
	case "windows":
		cmd, args = "rundll32", []string{"url.dll,FileProtocolHandler"}
	default:
		cmd = "xdg-open"
	}
	return exec.Command(cmd, append(args, url)...).Start()
}

func openBrowserCmd(url string) tea.Cmd {
	return func() tea.Msg {
		return browserOpenedMsg{err: openBrowser(url)}
	}
}
