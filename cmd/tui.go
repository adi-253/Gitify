package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	// Color Palette - Spotify-inspired with modern aesthetics
	spotifyGreen   = lipgloss.Color("#1DB954")
	spotifyBlack   = lipgloss.Color("#121212")
	spotifyDark    = lipgloss.Color("#181818")
	spotifyGray    = lipgloss.Color("#282828")
	spotifyLight   = lipgloss.Color("#B3B3B3")
	accentPink     = lipgloss.Color("#E91E63")
	// accentPurple   = lipgloss.Color("#9B59B6")
	accentCyan     = lipgloss.Color("#00BCD4")
	// accentOrange   = lipgloss.Color("#FF9800")
	white          = lipgloss.Color("#FFFFFF")
	subtleGray     = lipgloss.Color("#404040")
	// highlightGreen = lipgloss.Color("#1ED760")

	// Gradient-like effect colors
	// gradientStart = lipgloss.Color("#667eea")
	// gradientEnd   = lipgloss.Color("#764ba2")

	// Main styles
	sectionHeader = lipgloss.NewStyle().
			Foreground(spotifyGreen).
			Bold(true).
			MarginBottom(1)

	// titleStyle = lipgloss.NewStyle().
	// 		Foreground(spotifyGreen).
	// 		Bold(true).
	// 		Padding(0, 1)

	logoStyle = lipgloss.NewStyle().
			Foreground(spotifyGreen).
			Bold(true)

	// statusStyle = lipgloss.NewStyle().
	// 		Foreground(spotifyLight).
	// 		Background(spotifyDark).
	// 		Padding(0, 2).
	// 		MarginTop(1)

	// Box styles with different themes
	sidebarBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(subtleGray).
			Padding(1, 2).
			Background(spotifyDark)

	contentBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(subtleGray).
			Padding(1, 2).
			Background(spotifyBlack)

	// Focused box style
	// focusedBoxStyle = lipgloss.NewStyle().
	// 		Border(lipgloss.RoundedBorder()).
	// 		BorderForeground(spotifyGreen).
	// 		Padding(1, 2).
	// 		Background(spotifyBlack)

	// Sidebar item styles
	sidebarItemStyle = lipgloss.NewStyle().
				Foreground(spotifyLight).
				PaddingLeft(2)

	// sidebarItemActiveStyle = lipgloss.NewStyle().
	// 			Foreground(white).
	// 			Background(spotifyGray).
	// 			Bold(true).
	// 			PaddingLeft(2).
	// 			PaddingRight(2)

	sidebarItemFocusedStyle = lipgloss.NewStyle().
				Foreground(spotifyBlack).
				Background(spotifyGreen).
				Bold(true).
				PaddingLeft(2).
				PaddingRight(2)

	// Help text style
	helpStyle = lipgloss.NewStyle().
			Foreground(subtleGray).
			Italic(true)

	// Now playing indicator
	nowPlayingStyle = lipgloss.NewStyle().
			Foreground(spotifyGreen).
			Bold(true)

	// Divider style
	dividerStyle = lipgloss.NewStyle().
			Foreground(subtleGray)

	// Search input style
	// searchBoxStyle = lipgloss.NewStyle().
	// 		Border(lipgloss.RoundedBorder()).
	// 		BorderForeground(accentCyan).
	// 		Padding(0, 1).
	// 		MarginBottom(1)

	// // Track info styles
	// trackTitleStyle = lipgloss.NewStyle().
	// 		Foreground(white).
	// 		Bold(true)

	// trackArtistStyle = lipgloss.NewStyle().
	// 			Foreground(spotifyLight)

	// Status indicators
	playingIndicatorStyle = lipgloss.NewStyle().
				Foreground(spotifyGreen).
				Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(accentPink).
			Bold(true)

	successStyle = lipgloss.NewStyle().
			Foreground(spotifyGreen)

	// Header decoration
	// headerDecorStyle = lipgloss.NewStyle().
	// 			Foreground(gradientStart)
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

// ---------- Custom List Delegate ----------

type customDelegate struct {
	showDesc   bool
	isPlaylist bool
}

func newCustomDelegate(showDesc, isPlaylist bool) customDelegate {
	return customDelegate{showDesc: showDesc, isPlaylist: isPlaylist}
}

func (d customDelegate) Height() int {
	if d.showDesc {
		return 2
	}
	return 1
}

func (d customDelegate) Spacing() int { return 0 }

func (d customDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }

func (d customDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	var title, desc string

	if i, ok := item.(trackRow); ok {
		title = i.Title()
		desc = i.Description()
	} else if i, ok := item.(playlistItem); ok {
		title = i.Title()
		desc = ""
	} else {
		return
	}

	selected := index == m.Index()

	// Icons
	playlistIcon := "📁"
	trackIcon := "♪"
	selectedIcon := "▶"

	icon := trackIcon
	if d.isPlaylist {
		icon = playlistIcon
	}

	// Styling based on selection state
	var titleStr string
	if selected {
		titleStyle := lipgloss.NewStyle().
			Foreground(spotifyBlack).
			Background(spotifyGreen).
			Bold(true).
			Padding(0, 1)
		titleStr = titleStyle.Render(fmt.Sprintf("%s %s", selectedIcon, title))
	} else {
		titleStyle := lipgloss.NewStyle().
			Foreground(white).
			Padding(0, 1)
		titleStr = titleStyle.Render(fmt.Sprintf("%s %s", icon, title))
	}

	fmt.Fprint(w, titleStr)

	if d.showDesc && desc != "" {
		descStyle := lipgloss.NewStyle().
			Foreground(spotifyLight).
			PaddingLeft(4)
		if selected {
			descStyle = descStyle.Foreground(subtleGray)
		}
		fmt.Fprintf(w, "\n%s", descStyle.Render(desc))
	}
}

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

type playbackUpdatedMsg struct {
	info *PlaybackInfo
}

// ---------- Init helpers ----------

func initialModel() tuiModel {
	ti := textinput.New()
	ti.Placeholder = "🔍 Search for tracks, artists, albums..."
	ti.Focus()
	ti.CharLimit = 128
	ti.Width = 40
	ti.PromptStyle = lipgloss.NewStyle().Foreground(accentCyan)
	ti.TextStyle = lipgloss.NewStyle().Foreground(white)
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(subtleGray).Italic(true)
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(spotifyGreen)

	sidebar := []string{"🎵 Now Playing", "📁 Playlists", "🔍 Search"}

	// Custom delegate for playlists (no description, playlist icons)
	playlistDelegate := newCustomDelegate(false, true)
	playlistList := list.New(nil, playlistDelegate, 0, 0)
	playlistList.Title = "📁 Playlists"
	playlistList.SetShowStatusBar(false)
	playlistList.SetFilteringEnabled(true)
	playlistList.SetShowHelp(false)
	playlistList.Styles.Title = lipgloss.NewStyle().
		Foreground(spotifyGreen).
		Bold(true).
		MarginBottom(1)
	playlistList.Styles.FilterPrompt = lipgloss.NewStyle().Foreground(accentCyan)
	playlistList.Styles.FilterCursor = lipgloss.NewStyle().Foreground(spotifyGreen)

	// Custom delegate for tracks (with description/artist)
	trackDelegate := newCustomDelegate(true, false)
	trackList := list.New(nil, trackDelegate, 0, 0)
	trackList.Title = "♪ Tracks"
	trackList.SetShowStatusBar(false)
	trackList.SetFilteringEnabled(false)
	trackList.SetShowHelp(false)
	trackList.Styles.Title = lipgloss.NewStyle().
		Foreground(spotifyGreen).
		Bold(true).
		MarginBottom(1)

	// Custom delegate for search results (with description)
	searchDelegate := newCustomDelegate(true, false)
	searchList := list.New(nil, searchDelegate, 0, 0)
	searchList.Title = "🔍 Search Results"
	searchList.SetShowStatusBar(false)
	searchList.SetFilteringEnabled(false)
	searchList.SetShowHelp(false)
	searchList.Styles.Title = lipgloss.NewStyle().
		Foreground(accentCyan).
		Bold(true).
		MarginBottom(1)

	return tuiModel{
		keys:            defaultKeyMap(),
		status:          "✨ Welcome to Gitify TUI · Loading profile…",
		focus:           focusSidebar,
		sidebarSections: sidebar,
		playlistList:    playlistList,
		trackList:       trackList,
		searchInput:     ti,
		searchList:      searchList,
		lastActionAt:    time.Now(),
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

func fetchPlaybackCmd() tea.Cmd {
	return tea.Tick(time.Millisecond*500, func(t time.Time) tea.Msg {
		info, err := GetCurrentPlayback()
		if err != nil {
			return nil // silently ignore errors
		}
		return playbackUpdatedMsg{info: info}
	})
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
			m.status = "🔐 Not logged in · Run `gitify spotify login` first"
		} else {
			if m.userProfile != nil {
				m.status = fmt.Sprintf("👋 Hello, %s · Loading playlists…", m.userProfile.Username)
				cmds = append(cmds, loadPlaylistsCmd(m.userProfile.Userid))
			} else {
				m.status = "✅ Logged in · Loading playlists…"
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
			m.status = "📭 No playlists found"
		} else {
			m.status = fmt.Sprintf("📁 Loaded %d playlists", len(items))
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
			m.status = "📭 This playlist has no tracks"
		} else {
			m.status = fmt.Sprintf("🎵 Loaded %d tracks", len(items))
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
			m.status = fmt.Sprintf("🔍 No results for %q", msg.query)
		} else {
			m.status = fmt.Sprintf("🔍 Found %d tracks for %q", len(items), msg.query)
		}
		m.focus = focusSearchResults
	case errMsg:
		m.errMsg = msg.Error()
		m.status = "❌ Error: " + msg.Error()
	case playbackUpdatedMsg:
		if msg.info != nil {
			m.isPlaying = msg.info.IsPlaying
			if msg.info.TrackName != "" {
				m.currentTrackURI = msg.info.TrackURI
				if m.isPlaying {
					m.status = fmt.Sprintf("🎵 Playing: %s — %s", msg.info.TrackName, msg.info.ArtistName)
				} else {
					m.status = fmt.Sprintf("⏸ Paused: %s — %s", msg.info.TrackName, msg.info.ArtistName)
				}
			}
		}
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
				m.status = "📭 No playlists loaded yet"
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
				m.status = "⏸ Paused"
			} else {
				go ResumePlayback()
				m.isPlaying = true
				m.status = "▶ Resumed"
			}
			m.lastActionAt = time.Now()
			return m, fetchPlaybackCmd()
		case key.Matches(msg, m.keys.Next):
			go NextTrack()
			m.lastActionAt = time.Now()
			m.status = "⏭ Skipping to next..."
			return m, fetchPlaybackCmd()
		case key.Matches(msg, m.keys.Prev):
			go PreviousTrack()
			m.lastActionAt = time.Now()
			m.status = "⏮ Going to previous..."
			return m, fetchPlaybackCmd()
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
				m.status = "⏳ Loading tracks…"
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
				m.status = "🔍 Searching…"
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
		m.status = "⚠️ Track URI not available"
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
	m.status = fmt.Sprintf("🎵 Playing: %s — %s", track.Name, joinArtists(track.Artists))
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
		m.status = "⚠️ Track URI not available"
		return
	}
	uris := []string{track.URI}
	go StartMusic(nil, &uris)
	m.isPlaying = true
	m.currentTrackURI = track.URI
	m.lastActionAt = time.Now()
	m.status = fmt.Sprintf("🎵 Playing: %s — %s", track.Name, m.getSearchArtistNames(track.Artists))
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
		return logoStyle.Render("  🎵 Loading Gitify TUI…")
	}

	sidebarWidth := 24
	contentWidth := m.width - sidebarWidth - 6

	// Determine which box style to use based on focus
	sidebarStyle := sidebarBoxStyle
	contentStyle := contentBoxStyle

	if m.focus == focusSidebar {
		sidebarStyle = sidebarStyle.BorderForeground(spotifyGreen)
	}
	if m.focus == focusPlaylists || m.focus == focusTracks || m.focus == focusSearch || m.focus == focusSearchResults {
		contentStyle = contentStyle.BorderForeground(spotifyGreen)
	}

	sidebar := m.renderSidebar(sidebarWidth - 4)
	content := m.renderContent(contentWidth - 4)
	status := m.renderStatusBar()

	main := lipgloss.JoinHorizontal(
		lipgloss.Top,
		sidebarStyle.Width(sidebarWidth).Height(m.height-4).Render(sidebar),
		contentStyle.Width(contentWidth).Height(m.height-4).Render(content),
	)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		main,
		status,
	)
}

func (m tuiModel) renderSidebar(width int) string {
	var rows []string

	// ASCII Art Logo
	logo := `
 ██████╗ ██╗████████╗██╗███████╗██╗   ██╗
██╔════╝ ██║╚══██╔══╝██║██╔════╝╚██╗ ██╔╝
██║  ███╗██║   ██║   ██║█████╗   ╚████╔╝ 
██║   ██║██║   ██║   ██║██╔══╝    ╚██╔╝  
╚██████╔╝██║   ██║   ██║██║        ██║   
 ╚═════╝ ╚═╝   ╚═╝   ╚═╝╚═╝        ╚═╝`

	// Compact logo for narrow sidebars
	compactLogo := logoStyle.Render("🎵 Gitify")
	if width > 45 {
		rows = append(rows, logoStyle.Render(logo))
	} else {
		rows = append(rows, compactLogo)
	}

	rows = append(rows, "")
	rows = append(rows, dividerStyle.Render(strings.Repeat("─", width)))
	rows = append(rows, "")

	// Menu items with better styling
	for i, s := range m.sidebarSections {
		var itemStyle lipgloss.Style

		if focusSidebar == m.focus && i == m.sidebarIndex {
			itemStyle = sidebarItemFocusedStyle
		} else if i == 0 && m.isPlaying {
			itemStyle = nowPlayingStyle
		} else {
			itemStyle = sidebarItemStyle
		}

		line := itemStyle.Render(s)
		rows = append(rows, line)
	}

	rows = append(rows, "")
	rows = append(rows, dividerStyle.Render(strings.Repeat("─", width)))
	rows = append(rows, "")

	// Help section with better formatting
	helpLines := []string{
		"⌨️  Shortcuts",
		"",
		helpStyle.Render("  /,s  Search"),
		helpStyle.Render("  p    Playlists"),
		helpStyle.Render("  ␣    Play/Pause"),
		helpStyle.Render("  ←/→  Prev/Next"),
		helpStyle.Render("  q    Quit"),
	}

	for _, line := range helpLines {
		rows = append(rows, line)
	}

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
	if colWidth < 28 {
		colWidth = 28
	}

	// Adjust lists to size
	m.playlistList.SetWidth(colWidth - 4)
	m.playlistList.SetHeight(m.height - 10)
	m.trackList.SetWidth(colWidth - 4)
	m.trackList.SetHeight(m.height - 10)

	// Mark focus with styled titles
	if m.focus == focusPlaylists {
		m.playlistList.Title = "▶ 📁 Playlists"
		m.playlistList.Styles.Title = lipgloss.NewStyle().
			Foreground(spotifyGreen).
			Bold(true).
			Background(spotifyGray).
			Padding(0, 1).
			MarginBottom(1)
	} else {
		m.playlistList.Title = "📁 Playlists"
		m.playlistList.Styles.Title = lipgloss.NewStyle().
			Foreground(spotifyLight).
			Bold(true).
			MarginBottom(1)
	}

	if m.focus == focusTracks {
		m.trackList.Title = "▶ ♪ Tracks"
		m.trackList.Styles.Title = lipgloss.NewStyle().
			Foreground(spotifyGreen).
			Bold(true).
			Background(spotifyGray).
			Padding(0, 1).
			MarginBottom(1)
	} else {
		m.trackList.Title = "♪ Tracks"
		m.trackList.Styles.Title = lipgloss.NewStyle().
			Foreground(spotifyLight).
			Bold(true).
			MarginBottom(1)
	}

	// Add playlist name to tracks title if selected
	if m.currentPlaylistIdx >= 0 && m.currentPlaylistIdx < len(m.playlists) {
		playlistName := m.playlists[m.currentPlaylistIdx].Name
		if len(playlistName) > 20 {
			playlistName = playlistName[:17] + "..."
		}
		if m.focus == focusTracks {
			m.trackList.Title = fmt.Sprintf("▶ ♪ %s", playlistName)
		} else {
			m.trackList.Title = fmt.Sprintf("♪ %s", playlistName)
		}
	}

	// Style the columns with subtle borders
	leftStyle := lipgloss.NewStyle().
		Width(colWidth).
		BorderRight(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(subtleGray).
		PaddingRight(1)

	rightStyle := lipgloss.NewStyle().
		Width(colWidth).
		PaddingLeft(1)

	left := leftStyle.Render(m.playlistList.View())
	right := rightStyle.Render(m.trackList.View())

	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

func (m tuiModel) renderSearch(width int) string {
	var sections []string

	// Search header
	searchTitle := sectionHeader.Render("🔍 Search")
	if m.focus == focusSearch {
		searchTitle = lipgloss.NewStyle().
			Foreground(spotifyBlack).
			Background(accentCyan).
			Bold(true).
			Padding(0, 1).
			Render("▶ 🔍 Search")
	}
	sections = append(sections, searchTitle)

	// Styled search input box
	inputStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(subtleGray).
		Padding(0, 1).
		Width(width - 4)

	if m.focus == focusSearch {
		inputStyle = inputStyle.BorderForeground(accentCyan)
	}

	sections = append(sections, inputStyle.Render(m.searchInput.View()))
	sections = append(sections, "")

	// Search results
	m.searchList.SetWidth(width - 4)
	m.searchList.SetHeight(m.height - 14)

	if m.focus == focusSearchResults {
		m.searchList.Title = "▶ 🔍 Search Results"
		m.searchList.Styles.Title = lipgloss.NewStyle().
			Foreground(accentCyan).
			Bold(true).
			Background(spotifyGray).
			Padding(0, 1).
			MarginBottom(1)
	} else {
		m.searchList.Title = "🔍 Search Results"
		m.searchList.Styles.Title = lipgloss.NewStyle().
			Foreground(spotifyLight).
			Bold(true).
			MarginBottom(1)
	}

	sections = append(sections, m.searchList.View())

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m tuiModel) renderStatusBar() string {
	// Left side: playback status
	var leftParts []string

	if m.isPlaying {
		playIcon := playingIndicatorStyle.Render("▶ Now Playing")
		leftParts = append(leftParts, playIcon)
	} else {
		pauseIcon := lipgloss.NewStyle().Foreground(spotifyLight).Render("⏸ Paused")
		leftParts = append(leftParts, pauseIcon)
	}

	// Status message
	if m.status != "" {
		statusMsg := m.status
		if strings.HasPrefix(statusMsg, "Error") {
			statusMsg = errorStyle.Render("⚠ " + statusMsg)
		} else if strings.Contains(statusMsg, "Playing") {
			statusMsg = successStyle.Render("🎵 " + statusMsg)
		} else {
			statusMsg = lipgloss.NewStyle().Foreground(spotifyLight).Render(statusMsg)
		}
		leftParts = append(leftParts, statusMsg)
	}

	leftContent := strings.Join(leftParts, " │ ")

	// Right side: controls hint
	rightContent := helpStyle.Render("␣: Play/Pause  ←→: Skip  q: Quit")

	// Calculate spacing
	leftLen := lipgloss.Width(leftContent)
	rightLen := lipgloss.Width(rightContent)
	spacerLen := m.width - leftLen - rightLen - 6
	if spacerLen < 1 {
		spacerLen = 1
	}

	spacer := strings.Repeat(" ", spacerLen)

	fullStatus := leftContent + spacer + rightContent

	// Keep status bar to a single line
	if m.width > 0 {
		runes := []rune(fullStatus)
		max := m.width - 4
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
			fullStatus = string(runes)
		}
	}

	return lipgloss.NewStyle().
		Background(spotifyDark).
		Foreground(spotifyLight).
		Padding(0, 2).
		Width(m.width).
		Render(fullStatus)
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