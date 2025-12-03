package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/adi-253/Gitify/cmd/utils"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

// ---------- Styling ----------

var (
	primaryColor  = lipgloss.Color("#7DF9FF") // neon cyan
	bgColor       = lipgloss.Color("#050608")
	borderFg      = lipgloss.Color("#3C3F51")
	dimColor      = lipgloss.Color("#7B7D8A")
	playingColor  = lipgloss.Color("#9AEDFE")
	sectionHeader = lipgloss.NewStyle().Foreground(primaryColor).Bold(true)
	titleStyle    = lipgloss.NewStyle().Foreground(primaryColor).Bold(true).Padding(0, 1)
	statusStyle   = lipgloss.NewStyle().Foreground(dimColor).Background(bgColor).Padding(0, 1)
	boxStyle      = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(borderFg).Padding(0, 1).Margin(0, 1)
)

// ---------- Keymap ----------

type keyMap struct {
	Quit      key.Binding
	Help      key.Binding
	NextPane  key.Binding
	PrevPane  key.Binding
	Search    key.Binding
	Playlists key.Binding
	Play      key.Binding
	Pause     key.Binding
	Next      key.Binding
	Prev      key.Binding
}

func defaultKeyMap() keyMap {
	return keyMap{
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("h", "?"),
			key.WithHelp("h/?", "toggle help"),
		),
		NextPane: key.NewBinding(
			key.WithKeys("]"),
			key.WithHelp("]", "next pane"),
		),
		PrevPane: key.NewBinding(
			key.WithKeys("["),
			key.WithHelp("[", "prev pane"),
		),
		Search: key.NewBinding(
			key.WithKeys("/", "s"),
			key.WithHelp("/, s", "focus search"),
		),
		Playlists: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "focus playlists"),
		),
		Play: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "play"),
		),
		Pause: key.NewBinding(
			key.WithKeys(" "),
			key.WithHelp("space", "play/pause"),
		),
		Next: key.NewBinding(
			key.WithKeys("l", "right"),
			key.WithHelp("→/l", "next track"),
		),
		Prev: key.NewBinding(
			key.WithKeys("h", "left"),
			key.WithHelp("←/h", "prev track"),
		),
	}
}

// ---------- Bubble Tea model ----------

type focusArea int

const (
	focusSidebar focusArea = iota
	focusPlaylists
	focusTracks
	focusSearch
	focusSearchResults
)

type trackRow struct {
	title  string
	sub    string
	isFrom string // "playlist" or "search"
	index  int
}

func (t trackRow) Title() string       { return t.title }
func (t trackRow) Description() string { return t.sub }
func (t trackRow) FilterValue() string { return t.title + " " + t.sub }

type playlistItem struct {
	name string
}

func (p playlistItem) Title() string       { return p.name }
func (p playlistItem) Description() string { return "" }
func (p playlistItem) FilterValue() string { return p.name }

type tuiModel struct {
	width  int
	height int

	keys   keyMap
	status string

	focus focusArea

	// login/profile
	isLoggedIn  bool
	userProfile *Profile

	// data
	playlists     []Playlist
	currentPlaylistIdx int
	currentTracks  []PlaylistTrack
	searchTracks   []TrackItem

	// UI components
	sidebarSections []string
	sidebarIndex    int
	playlistList    list.Model
	trackList       list.Model
	searchInput     textinput.Model
	searchList      list.Model

	// playback
	isPlaying       bool
	currentTrackURI string
	lastActionAt    time.Time

	loading bool
	errMsg  string
}

// ---------- Messages ----------

type errMsg error

type playlistsLoadedMsg struct {
	playlists []Playlist
}

type tracksLoadedMsg struct {
	playlistIdx int
	tracks      []PlaylistTrack
}

type searchResultsMsg struct {
	tracks []TrackItem
	query  string
}

type profileLoadedMsg struct {
	profile *Profile
	logged  bool
}

// ---------- Init helpers ----------

func initialModel() tuiModel {
	ti := textinput.New()
	ti.Placeholder = "Search tracks..."
	ti.Focus()
	ti.CharLimit = 128

	sidebar := []string{"Now Playing", "Playlists", "Search", "Profile", "Controls"}

	playlistList := list.New(nil, list.NewDefaultDelegate(), 0, 0)
	playlistList.Title = "Playlists"
	playlistList.SetShowStatusBar(false)
	playlistList.SetFilteringEnabled(true)
	playlistList.SetShowHelp(false)

	trackList := list.New(nil, list.NewDefaultDelegate(), 0, 0)
	trackList.Title = "Tracks"
	trackList.SetShowStatusBar(false)
	trackList.SetFilteringEnabled(false)
	trackList.SetShowHelp(false)

	searchList := list.New(nil, list.NewDefaultDelegate(), 0, 0)
	searchList.Title = "Search Results"
	searchList.SetShowStatusBar(false)
	searchList.SetFilteringEnabled(false)
	searchList.SetShowHelp(false)

	return tuiModel{
		keys:  defaultKeyMap(),
		status: "Welcome to Gitify TUI · Loading profile…",
		focus: focusSidebar,
		sidebarSections: sidebar,
		playlistList:   playlistList,
		trackList:      trackList,
		searchInput:    ti,
		searchList:     searchList,
		lastActionAt:   time.Now(),
	}
}

// ---------- Async loaders ----------

func loadProfileCmd() tea.Cmd {
	return func() tea.Msg {
		// Check if user is logged in by looking for token.json
		if _, err := os.Stat("token.json"); err != nil {
			return profileLoadedMsg{profile: nil, logged: false}
		}
		data, err := os.ReadFile("profile.json")
		if err != nil {
			return profileLoadedMsg{profile: nil, logged: true}
		}
		var p Profile
		if json.Unmarshal(data, &p) != nil {
			return profileLoadedMsg{profile: nil, logged: true}
		}
		return profileLoadedMsg{profile: &p, logged: true}
	}
}

func loadPlaylistsCmd(userID string) tea.Cmd {
	return func() tea.Msg {
		client, err := utils.NewSpotifyClient()
		if err != nil {
			return errMsg(err)
		}

		urlStr := "https://api.spotify.com/v1/users/" + userID + "/playlists"
		resp, err := client.Get(urlStr)
		if err != nil {
			return errMsg(err)
		}
		defer resp.Body.Close()

		var playlistsResp PlaylistsResponse
		if err := json.NewDecoder(resp.Body).Decode(&playlistsResp); err != nil {
			return errMsg(err)
		}

		return playlistsLoadedMsg{playlists: playlistsResp.Items}
	}
}

func loadTracksCmd(p Playlist, idx int) tea.Cmd {
	return func() tea.Msg {
		client, err := utils.NewSpotifyClient()
		if err != nil {
			return errMsg(err)
		}

		var all []PlaylistTrack
		next := p.Tracks.Href

		for next != "" {
			resp, err := client.Get(next)
		if err != nil {
				return errMsg(err)
		}
		defer resp.Body.Close()
		
			var res PlaylistTracksResponse
			if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
				return errMsg(err)
			}
			all = append(all, res.Items...)
			next = res.Next
		}

		return tracksLoadedMsg{playlistIdx: idx, tracks: all}
	}
}

func searchCmd(query string) tea.Cmd {
	return func() tea.Msg {
		client, err := utils.NewSpotifyClient()
		if err != nil {
			return errMsg(err)
		}
		
		baseURL, err := url.Parse("https://api.spotify.com/v1/search")
		if err != nil {
			return errMsg(err)
		}
		params := url.Values{}
		params.Add("q", query)
		params.Add("type", "track")
		params.Add("limit", "30")
		baseURL.RawQuery = params.Encode()
		
		resp, err := client.Get(baseURL.String())
		if err != nil {
			return errMsg(err)
		}
		defer resp.Body.Close()
		
		var result SearchResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return errMsg(err)
		}

		return searchResultsMsg{tracks: result.Tracks.Items, query: query}
	}
}

// ---------- Bubble Tea interface ----------

func (m tuiModel) Init() tea.Cmd {
	return tea.Batch(
		loadProfileCmd(),
		tea.ClearScreen,
	)
}

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case profileLoadedMsg:
		m.isLoggedIn = msg.logged
		m.userProfile = msg.profile
		if !m.isLoggedIn {
			m.status = "Not logged in · Run `gitify spotify login` and then open TUI"
		} else {
			if m.userProfile != nil {
				m.status = fmt.Sprintf("Hello, %s · Loading playlists…", m.userProfile.Username)
				cmds = append(cmds, loadPlaylistsCmd(m.userProfile.Userid))
			} else {
				m.status = "Logged in · Loading playlists…"
				// fallback user info
				cmds = append(cmds, loadPlaylistsCmd("me"))
			}
		}
	case playlistsLoadedMsg:
		m.playlists = msg.playlists
		items := make([]list.Item, len(msg.playlists))
		for i, p := range msg.playlists {
			items[i] = playlistItem{name: fmt.Sprintf("%s", p.Name)}
		}
		m.playlistList.SetItems(items)
		if len(items) == 0 {
			m.status = "No playlists found"
		} else {
			m.status = fmt.Sprintf("Loaded %d playlists", len(items))
		}
	case tracksLoadedMsg:
		if msg.playlistIdx < 0 || msg.playlistIdx >= len(m.playlists) {
			break
		}
		m.currentPlaylistIdx = msg.playlistIdx
		m.currentTracks = msg.tracks
		items := make([]list.Item, 0, len(msg.tracks))
		for i, t := range msg.tracks {
			sub := joinArtists(t.Track.Artists)
			items = append(items, trackRow{
				title:  t.Track.Name,
				sub:    sub,
				isFrom: "playlist",
				index:  i,
			})
		}
		m.trackList.SetItems(items)
		if len(items) == 0 {
			m.status = "This playlist has no tracks"
		} else {
			m.status = fmt.Sprintf("Loaded %d tracks from playlist", len(items))
		}
		m.focus = focusTracks
	case searchResultsMsg:
		m.searchTracks = msg.tracks
		items := make([]list.Item, 0, len(msg.tracks))
		for i, t := range msg.tracks {
			sub := m.getSearchArtistNames(t.Artists)
			items = append(items, trackRow{
				title:  t.Name,
				sub:    sub,
				isFrom: "search",
				index:  i,
			})
		}
		m.searchList.SetItems(items)
		if len(items) == 0 {
			m.status = fmt.Sprintf("No results for %q", msg.query)
		} else {
			m.status = fmt.Sprintf("Found %d tracks for %q", len(items), msg.query)
		}
		m.focus = focusSearchResults
	case errMsg:
		m.errMsg = msg.Error()
		m.status = "Error: " + msg.Error()
	case tea.KeyMsg:
		if key.Matches(msg, m.keys.Quit) {
			return m, tea.Quit
		}

		if key.Matches(msg, m.keys.Search) && m.focus != focusSearch {
			m.focus = focusSearch
			m.searchInput.Focus()
			return m, nil
		}

		if key.Matches(msg, m.keys.Playlists) {
			if len(m.playlistList.Items()) == 0 {
				m.status = "No playlists loaded yet"
				return m, nil
			}
			m.focus = focusPlaylists
			return m, nil
		}

		// When typing in the search box, don't steal normal characters for
		// playback/navigation shortcuts – let the text input handle them.
		if m.focus == focusSearch {
			break
		}

		switch {
		case key.Matches(msg, m.keys.Pause):
			if m.isPlaying {
				go PausePlayback()
				m.isPlaying = false
			} else {
				go ResumePlayback()
				m.isPlaying = true
			}
			m.lastActionAt = time.Now()
			return m, nil
		case key.Matches(msg, m.keys.Next):
			go NextTrack()
			m.lastActionAt = time.Now()
			return m, nil
		case key.Matches(msg, m.keys.Prev):
			go PreviousTrack()
			m.lastActionAt = time.Now()
			return m, nil
		}
	}

	// delegate to focused components
	switch m.focus {
	case focusPlaylists:
		var cmd tea.Cmd
		m.playlistList, cmd = m.playlistList.Update(msg)
		if km, ok := msg.(tea.KeyMsg); ok && key.Matches(km, m.keys.Play) {
			if len(m.playlists) == 0 {
				break
			}
			idx := m.playlistList.Index()
			if idx >= 0 && idx < len(m.playlists) {
				m.status = "Loading tracks…"
				cmds = append(cmds, loadTracksCmd(m.playlists[idx], idx))
			}
		}
		cmds = append(cmds, cmd)
	case focusTracks:
		var cmd tea.Cmd
		m.trackList, cmd = m.trackList.Update(msg)
		if km, ok := msg.(tea.KeyMsg); ok && key.Matches(km, m.keys.Play) {
			m.playSelectedTrackFromList()
		}
		cmds = append(cmds, cmd)
	case focusSearch:
		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)
		if km, ok := msg.(tea.KeyMsg); ok && km.Type == tea.KeyEnter {
			q := strings.TrimSpace(m.searchInput.Value())
			if q != "" {
				m.status = "Searching…"
				cmds = append(cmds, searchCmd(q))
			}
		}
		cmds = append(cmds, cmd)
	case focusSearchResults:
		var cmd tea.Cmd
		m.searchList, cmd = m.searchList.Update(msg)
		if km, ok := msg.(tea.KeyMsg); ok && key.Matches(km, m.keys.Play) {
			m.playSelectedSearchTrackFromList()
		}
		cmds = append(cmds, cmd)
	default:
		// sidebar focus: nothing special yet
	}

	return m, tea.Batch(cmds...)
}

func (m *tuiModel) cycleFocus() {
	switch m.focus {
	case focusSidebar:
		if len(m.playlists) > 0 {
			m.focus = focusPlaylists
			return
		}
		if len(m.currentTracks) > 0 {
			m.focus = focusTracks
			return
		}
		m.focus = focusSearch
	case focusPlaylists:
		if len(m.currentTracks) > 0 {
			m.focus = focusTracks
		} else {
			m.focus = focusSearch
		}
	case focusTracks:
		m.focus = focusSearch
	case focusSearch:
		if len(m.searchTracks) > 0 {
			m.focus = focusSearchResults
		} else {
			m.focus = focusSidebar
		}
	case focusSearchResults:
		m.focus = focusSidebar
	}
}

// playback helpers using existing playback.go functions

func (m *tuiModel) playSelectedTrackFromList() {
	if len(m.currentTracks) == 0 {
		return
	}
	idx := m.trackList.Index()
	if idx < 0 || idx >= len(m.currentTracks) {
		return
	}
	track := m.currentTracks[idx].Track
	if track.URI == "" {
		m.status = "Track URI not available"
		return
	}
	// Prefer playlist context when possible
	if m.currentPlaylistIdx >= 0 && m.currentPlaylistIdx < len(m.playlists) {
		pl := m.playlists[m.currentPlaylistIdx]
		playlistURI := pl.Uri
		if playlistURI == "" {
			playlistURI = "spotify:playlist:" + pl.ID
		}
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = ctx // currently unused, but ready for extensions
			StartMusic(&playlistURI, nil)
		}()
	} else {
		uris := []string{track.URI}
		go StartMusic(nil, &uris)
	}
	m.isPlaying = true
	m.currentTrackURI = track.URI
	m.lastActionAt = time.Now()
	m.status = fmt.Sprintf("Playing %s — %s", track.Name, joinArtists(track.Artists))
}

func (m *tuiModel) playSelectedSearchTrackFromList() {
	if len(m.searchTracks) == 0 {
		return
	}
	idx := m.searchList.Index()
	if idx < 0 || idx >= len(m.searchTracks) {
		return
	}
	track := m.searchTracks[idx]
	if track.URI == "" {
		m.status = "Track URI not available"
		return
	}
	uris := []string{track.URI}
	go StartMusic(nil, &uris)
	m.isPlaying = true
	m.currentTrackURI = track.URI
	m.lastActionAt = time.Now()
	m.status = fmt.Sprintf("Playing %s — %s", track.Name, m.getSearchArtistNames(track.Artists))
}

// artist helpers (reused logic from old TUI)
func (m *tuiModel) getSearchArtistNames(artists []ArtistResp) string {
	var names []string
	for _, a := range artists {
		names = append(names, a.Name)
	}
	return strings.Join(names, ", ")
}

// ---------- View ----------

func (m tuiModel) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading Gitify TUI…"
	}

	sidebarWidth := 20
	contentWidth := m.width - sidebarWidth - 4

	sidebar := m.renderSidebar(sidebarWidth - 2)
	content := m.renderContent(contentWidth - 2)
	status := m.renderStatusBar()

	main := lipgloss.JoinHorizontal(
		lipgloss.Top,
		boxStyle.Width(sidebarWidth).Render(sidebar),
		boxStyle.Width(contentWidth).Render(content),
	)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		main,
		status,
	)
}

func (m tuiModel) renderSidebar(width int) string {
	var rows []string

	title := titleStyle.Render("Gitify")
	rows = append(rows, title, "")

	for i, s := range m.sidebarSections {
		line := s
		if i == 0 && m.isPlaying {
			line = fmt.Sprintf("♪ %s", s)
		}
		st := lipgloss.NewStyle().Foreground(dimColor)
		if focusSidebar == m.focus && i == m.sidebarIndex {
			st = st.Foreground(primaryColor).Bold(true)
		}
		rows = append(rows, st.Render(line))
	}

	rows = append(rows, "", lipgloss.NewStyle().Foreground(dimColor).Render("Tab: cycle focus"))
	rows = append(rows, lipgloss.NewStyle().Foreground(dimColor).Render("/: search · h: help · q: quit"))

	return lipgloss.NewStyle().Width(width).Render(strings.Join(rows, "\n"))
}

func (m tuiModel) renderContent(width int) string {
	switch m.focus {
	case focusPlaylists, focusSidebar:
		return m.renderPlaylistsAndTracks(width)
	case focusTracks:
		return m.renderPlaylistsAndTracks(width)
	case focusSearch, focusSearchResults:
		return m.renderSearch(width)
	default:
		return m.renderPlaylistsAndTracks(width)
	}
}

func (m tuiModel) renderPlaylistsAndTracks(width int) string {
	colWidth := width / 2
	if colWidth < 24 {
		colWidth = 24
	}

	// adjust lists to size
	m.playlistList.SetWidth(colWidth - 4)
	m.playlistList.SetHeight(m.height - 6)
	m.trackList.SetWidth(colWidth - 4)
	m.trackList.SetHeight(m.height - 6)

	// mark focus
	plTitle := "Playlists"
	if m.focus == focusPlaylists {
		plTitle = "▶ Playlists"
	}
	m.playlistList.Title = plTitle

	trTitle := "Tracks"
	if m.focus == focusTracks {
		trTitle = "▶ Tracks"
	}
	if m.currentPlaylistIdx >= 0 && m.currentPlaylistIdx < len(m.playlists) {
		trTitle = fmt.Sprintf("%s · %s", trTitle, m.playlists[m.currentPlaylistIdx].Name)
	}
	m.trackList.Title = trTitle

	left := m.playlistList.View()
	right := m.trackList.View()

	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		lipgloss.NewStyle().Width(colWidth).Render(left),
		lipgloss.NewStyle().Width(colWidth).Render(right),
	)
}

func (m tuiModel) renderSearch(width int) string {
	searchBox := m.searchInput.View()
	if m.focus == focusSearch {
		searchBox = sectionHeader.Render("Search") + "\n" + searchBox
	} else {
		searchBox = "Search\n" + searchBox
	}

	m.searchList.SetWidth(width - 4)
	m.searchList.SetHeight(m.height - 8)
	if m.focus == focusSearchResults {
		m.searchList.Title = "▶ Search Results"
	} else {
		m.searchList.Title = "Search Results"
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		searchBox,
		"",
		m.searchList.View(),
	)
}

func (m tuiModel) renderStatusBar() string {
	var pieces []string
	if m.isPlaying {
		pieces = append(pieces, lipgloss.NewStyle().Foreground(playingColor).Render("♪ Playing"))
	}
	if m.status != "" {
		pieces = append(pieces, m.status)
	}

	line := strings.Join(pieces, " · ")

	// Keep status bar to a single line so it doesn't corrupt the TUI layout
	if m.width > 0 {
		runes := []rune(line)
		max := m.width - 2
		if max < 0 {
			max = 0
		}
		if len(runes) > max {
			ellipsis := []rune("…")
			if max > len(ellipsis) {
				runes = append(runes[:max-len(ellipsis)], ellipsis...)
			} else {
				runes = runes[:max]
			}
			line = string(runes)
		}
	}

	return statusStyle.Width(m.width).Render(line)
}

// TUI command
var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch the Terminal User Interface",
	Long:  "Launch GitifyTUI - A Bubble Tea powered terminal interface for Spotify, inspired by Lazygit and Charmbracelet UIs.",
	Run: func(cmd *cobra.Command, args []string) {
		// Avoid stdout prints from playback helpers so Bubble Tea layout
		// stays intact (no random lines injected into the UI).
		SetPlaybackSilent(true)

		p := tea.NewProgram(initialModel(), tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	spotifyCmd.AddCommand(tuiCmd)
}