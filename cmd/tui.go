package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/adi-253/Gitify/cmd/utils"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/spf13/cobra"
)

// TUIApp represents the main TUI application
type TUIApp struct {
	app         *tview.Application
	pages       *tview.Pages
	sidebar     *tview.List
	mainContent *tview.Flex
	statusBar   *tview.TextView
	helpModal   *tview.Modal
	
	// Content panels
	playlistList   *tview.List
	trackList      *tview.Table
	profileView    *tview.TextView
	searchInput    *tview.InputField
	searchResults  *tview.Table
	
	// State
	currentPanel   string
	isLoggedIn     bool
	userProfile    *Profile
	playlists      []Playlist
	currentTracks  []PlaylistTrack
	searchTracks   []TrackItem
	
	// Pagination state
	tracksPerPage     int
	currentPage       int
	totalPages        int
	currentPlaylist   string
	
	// Playback state
	isPlaying         bool
	currentTrackURI   string
}

// Colors and styles similar to Lazygit
var (
	primaryColor   = tcell.ColorBlue
	selectedColor  = tcell.ColorYellow
	errorColor     = tcell.ColorRed
	successColor   = tcell.ColorGreen
	borderColor    = tcell.ColorGray
	// infoColor      = tcell.ColorLightBlue
	warningColor   = tcell.ColorOrange
)

func NewTUIApp() *TUIApp {
	app := &TUIApp{
		app:          tview.NewApplication(),
		pages:        tview.NewPages(),
		currentPanel: "sidebar",
		tracksPerPage: 50, // Default tracks per page
		currentPage:   0,
		totalPages:    0,
	}
	
	app.checkLoginStatus()
	app.setupUI()
	app.setupKeybindings()
	
	return app
}

func (t *TUIApp) checkLoginStatus() {
	// Check if user is logged in by looking for token.json
	if _, err := os.Stat("token.json"); err == nil {
		t.isLoggedIn = true
		// Try to load profile data
		if data, err := os.ReadFile("profile.json"); err == nil {
			var profile Profile
			if json.Unmarshal(data, &profile) == nil {
				t.userProfile = &profile
			}
		}
	}
}

func (t *TUIApp) setupUI() {
	// Create sidebar
	t.sidebar = tview.NewList().
		SetMainTextColor(tcell.ColorWhite).
		SetSelectedTextColor(selectedColor).
		SetSelectedBackgroundColor(primaryColor)
	
	// Create main content area
	t.mainContent = tview.NewFlex().SetDirection(tview.FlexRow)
	
	// Create status bar
	t.statusBar = tview.NewTextView()
	t.statusBar.SetTextColor(tcell.ColorWhite)
	t.statusBar.SetBackgroundColor(primaryColor)
	t.statusBar.SetText(" GitifyTUI | Press 'h' for help | 'q' to quit ")
	
	// Setup sidebar items
	t.setupSidebar()
	
	// Show welcome screen initially
	t.showWelcomeScreen()
	
	// Create main layout
	mainFlex := tview.NewFlex().
		AddItem(t.sidebar, 30, 0, true).
		AddItem(t.mainContent, 0, 1, false)
	
	// Root layout with status bar
	rootFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(mainFlex, 0, 1, true).
		AddItem(t.statusBar, 1, 0, false)
	
	// Create help modal
	t.createHelpModal()
	
	// Add to pages
	t.pages.AddPage("main", rootFlex, true, true)
	t.pages.AddPage("help", t.helpModal, true, false)
	
	t.app.SetRoot(t.pages, true)
}

func (t *TUIApp) setupSidebar() {
	t.sidebar.Clear()
	
	// Add sidebar items based on login status
	if !t.isLoggedIn {
		t.sidebar.AddItem("L Login", "Login to Spotify", 'l', t.showLoginPanel)
		t.sidebar.AddItem("! Not Logged In", "Please login first", 0, nil)
	} else {
		profileName := "P Profile"
		if t.userProfile != nil {
			profileName = "P " + t.userProfile.Username
		}
		t.sidebar.AddItem(profileName, "User Profile", 'p', t.showProfilePanel)
		t.sidebar.AddItem("L Playlists", "View playlists", 'l', t.showPlaylistPanel)
		t.sidebar.AddItem("S Search", "Search songs", 's', t.showSearchPanel)
		t.sidebar.AddItem("- Pause", "Pause", 0, func() { go t.pausePlayback() })
		t.sidebar.AddItem("+ Resume", "Resume", 0, func() { go t.resumePlayback() })
		t.sidebar.AddItem("> Next", "Next track", 0, func() { go t.nextTrack() })
		t.sidebar.AddItem("< Prev", "Previous track", 0, func() { go t.previousTrack() })
		t.sidebar.AddItem("R Refresh", "Refresh data", 'r', t.refreshData)
		t.sidebar.AddItem("X Logout", "Logout", 0, t.logout)
	}
	
	t.sidebar.AddItem("H Help", "Show help", 'h', t.showHelp)
	t.sidebar.AddItem("Q Quit", "Exit", 'q', t.quit)
	
	// Set border and title
	t.sidebar.SetBorder(true).
		SetTitle(" Menu ").
		SetBorderColor(borderColor)
}

func (t *TUIApp) setupKeybindings() {
	t.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Don't intercept keys when in search input
		if t.currentPanel == "search" && t.app.GetFocus() == t.searchInput {
			switch event.Key() {
			case tcell.KeyEsc:
				t.focusSidebar()
				return nil
			case tcell.KeyTab:
				if t.searchResults != nil && t.searchResults.GetRowCount() > 0 {
					t.currentPanel = "search_results"
					t.app.SetFocus(t.searchResults)
				}
				return nil
			}
			return event
		}

		switch event.Key() {
		case tcell.KeyEsc:
			if t.pages.HasPage("help") {
				name, _ := t.pages.GetFrontPage()
				if name == "help" {
					t.pages.SwitchToPage("main")
					return nil
				}
			}
			t.focusSidebar()
			return nil
		case tcell.KeyTab:
			// Only allow Tab in specific contexts to avoid interfering with list navigation
			if t.currentPanel == "sidebar" || t.currentPanel == "search" {
				t.switchFocus()
			}
			return nil
		}
		
		// Handle pagination in tracks panel (check this first to avoid conflicts)
		if t.currentPanel == "tracks" {
			// Handle arrow keys and special keys first
			switch event.Key() {
			case tcell.KeyEnter:
				t.playSelectedTrack()
				return nil
			case tcell.KeyRune:
				if event.Rune() == ' ' {
					t.togglePlayback()
					return nil
				}
			case tcell.KeyRight, tcell.KeyPgDn:
				t.nextPage()
				return nil
			case tcell.KeyLeft, tcell.KeyPgUp:
				t.previousPage()
				return nil
			case tcell.KeyHome:
				t.firstPage()
				return nil
			case tcell.KeyEnd:
				t.lastPage()
				return nil
			}
			
			// Handle character keys
			switch event.Rune() {
			case 'n', '.', '>':
				t.nextPage()
				return nil
			case 'b', ',', '<':
				t.previousPage()
				return nil
			case 'g':
				t.firstPage()
				return nil
			case 'G':
				t.lastPage()
				return nil
			case 'h':
				t.showHelp()
				return nil
			case 'q':
				return event // Let it bubble up to quit if needed
			}
			// Let tview handle up/down navigation and other keys
			return event
		}
		
		// Handle playlist panel
		if t.currentPanel == "playlists" {
			switch event.Key() {
			case tcell.KeyEnter:
				// Let tview handle the selection (calls the selected function)
				return event
			}
			return event // Let tview handle up/down navigation
		}
		
		// Handle playback controls in search results
		if t.currentPanel == "search_results" {
			switch event.Key() {
			case tcell.KeyEnter:
				t.playSelectedSearchTrack()
				return nil
			case tcell.KeyRune:
				if event.Rune() == ' ' {
					t.togglePlayback()
					return nil
				}
			}
			return event // Let tview handle other keys (up/down navigation)
		}
		
		// Only process hotkeys when not in search results
		if t.currentPanel != "search_results" {
			switch event.Rune() {
			case 'q':
				if t.currentPanel == "sidebar" {
					t.quit()
					return nil
				}
			case 'h':
				t.showHelp()
				return nil
			case 'p':
				if t.isLoggedIn {
					t.showProfilePanel()
				}
				return nil
			case 'l':
				if t.isLoggedIn {
					t.showPlaylistPanel()
				} else {
					t.showLoginPanel()
				}
				return nil
			case 'r':
				if t.isLoggedIn {
					t.refreshData()
				}
				return nil
			case 's':
				if t.isLoggedIn {
					t.showSearchPanel()
				}
				return nil
			}
		}
		
		return event
	})
}

func (t *TUIApp) createHelpModal() {
	helpText := `
╭─────── GitifyTUI Help ───────╮

 Navigation:
   ↑/↓, j/k    Navigate lists
   Tab         Switch focus
   Esc         Focus sidebar
   Enter       Select item

 Actions:
   l           Login/View playlists  
   p           View profile
   s           Search for songs
   r           Refresh data
   h           Show this help
   q           Quit application

 In Playlists:
   Enter       View playlist tracks
   Esc         Back to playlists

 In Tracks (Pagination):
   →, n, .     Next page
   ←, b, ,     Previous page  
   Home, g     First page
   End, G      Last page
   PgUp/PgDn   Previous/Next page
   Enter       Play selected track
   Space       Play/Pause
   Esc         Back to playlists

 In Search:
   Enter       Perform search
   Tab         Switch to results
   Esc         Back to navigation
   
 In Search Results:
   Enter       Play selected track
   Space       Play/Pause
   Tab         Back to search

╰──────────────────────────────╯

Press Esc to close this help.
`
	
	t.helpModal = tview.NewModal().
		SetText(helpText).
		AddButtons([]string{"Close"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			t.pages.SwitchToPage("main")
		})
}

func (t *TUIApp) Run() error {
	return t.app.Run()
}

// Panel switching functions
func (t *TUIApp) focusSidebar() {
	t.currentPanel = "sidebar"
	t.app.SetFocus(t.sidebar)
	t.updateStatusBar()
}

func (t *TUIApp) switchFocus() {
	switch t.currentPanel {
	case "sidebar":
		if t.playlistList != nil && t.playlistList.GetItemCount() > 0 {
			t.currentPanel = "playlists"
			t.app.SetFocus(t.playlistList)
		} else if t.trackList != nil && t.trackList.GetRowCount() > 0 {
			t.currentPanel = "tracks"
			t.app.SetFocus(t.trackList)
		} else if t.searchInput != nil {
			t.currentPanel = "search"
			t.app.SetFocus(t.searchInput)
		} else if t.searchResults != nil && t.searchResults.GetRowCount() > 0 {
			t.currentPanel = "search_results"
			t.app.SetFocus(t.searchResults)
		}
	case "playlists":
		if t.trackList != nil && t.trackList.GetRowCount() > 0 {
			t.currentPanel = "tracks"
			t.app.SetFocus(t.trackList)
		} else {
			t.focusSidebar()
		}
	case "tracks":
		t.focusSidebar()
	case "search":
		if t.searchResults != nil && t.searchResults.GetRowCount() > 0 {
			t.currentPanel = "search_results"
			t.app.SetFocus(t.searchResults)
		} else {
			t.focusSidebar()
		}
	case "search_results":
		t.currentPanel = "search"
		t.app.SetFocus(t.searchInput)
	default:
		t.focusSidebar()
	}
	t.updateStatusBar()
}

func (t *TUIApp) updateStatusBar() {
	var text string
	switch t.currentPanel {
	case "sidebar":
		text = " Navigation | Tab: switch focus | Enter: select | h: help | q: quit "
	case "playlists": 
		text = " Playlists | Enter: view tracks | ↑↓: navigate | Esc: back "
	case "tracks":
		playStatus := ""
		if t.isPlaying {
			playStatus = " | ♪ Playing"
		}
		if t.totalPages > 1 {
			text = fmt.Sprintf(" Tracks (Page %d/%d) | Enter: play | Space: pause/play | ←→: navigate | n/b: next/back | g/G: first/last%s | Esc: back ", t.currentPage+1, t.totalPages, playStatus)
		} else {
			text = fmt.Sprintf(" Tracks | Enter: play | Space: pause/play%s | Esc: back ", playStatus)
		}
	case "search":
		text = " Search | Enter: search | Tab: switch to results | Esc: back "
	case "search_results":
		playStatus := ""
		if t.isPlaying {
			playStatus = " | ♪ Playing"
		}
		text = fmt.Sprintf(" Search Results | Enter: play | Space: pause/play | ↑↓: navigate%s | Esc: back ", playStatus)
	default:
		text = " GitifyTUI | Press 'h' for help | 'q' to quit "
	}
	
	if !t.isLoggedIn {
		text = " Not logged in | Press 'l' to login | h: help | q: quit "
	}
	
	t.statusBar.SetText(text)
}

func (t *TUIApp) showWelcomeScreen() {
	// Clear main content
	t.mainContent.Clear()
	
	var welcomeText string
	if t.isLoggedIn {
		username := "User"
		if t.userProfile != nil {
			username = t.userProfile.Username
		}
		welcomeText = fmt.Sprintf(`
╭──────────────────────────────────────╮
│            Welcome to GitifyTUI       │
│   A Beautiful Terminal UI for Spotify │
╰──────────────────────────────────────╯

👋 Hello, %s!

You're successfully logged in to Spotify.
Choose an option from the sidebar to get started:

🎵 View your playlists
🔍 Search for songs
👤 Check your profile
⏯️  Control playback  
🔄 Refresh your data
❓ Get help

Navigate with ↑↓ or j/k keys
Press Enter to play tracks or select items
Use Space to pause/resume playback
Press 's' for quick search access
`, username)
	} else {
		welcomeText = `
╭──────────────────────────────────────╮
│            Welcome to GitifyTUI       │
│   A Beautiful Terminal UI for Spotify │
╰──────────────────────────────────────╯

🔐 You need to login first!

To get started:
1. Press 'l' to see login instructions
2. Or run 'gitify spotify login' in terminal
3. Complete OAuth in your browser
4. Return here and press 'r' to refresh

Press 'h' for help or 'q' to quit
`
	}
	
	welcomeView := tview.NewTextView().
		SetText(welcomeText).
		SetTextColor(tcell.ColorWhite).
		SetBorder(true).
		SetTitle(" Welcome ").
		SetBorderColor(primaryColor)
	
	t.mainContent.AddItem(welcomeView, 0, 1, false)
}

// Action functions
func (t *TUIApp) showHelp() {
	t.pages.SwitchToPage("help")
}

func (t *TUIApp) quit() {
	t.app.Stop()
}

func (t *TUIApp) showLoginPanel() {
	if t.isLoggedIn {
		return
	}
	
	// Clear main content
	t.mainContent.Clear()
	
	loginText := tview.NewTextView().
		SetText(`🔐 Login to Spotify

To get started with Gitify:

1. Open a new terminal window/tab
2. Run: gitify spotify login
3. Complete OAuth in your browser  
4. Return here and press 'r' to refresh

Alternatively:
• Press 'q' to quit and use CLI mode
• Press Esc to go back to navigation

Status: Not logged in`).
		SetTextColor(tcell.ColorWhite).
		SetBorder(true).
		SetTitle(" Login Instructions ").
		SetBorderColor(warningColor)
	
	t.mainContent.AddItem(loginText, 0, 1, false)
	t.currentPanel = "login"
	t.updateStatusBar()
}

func (t *TUIApp) logout() {
	// Remove token and profile files
	os.Remove("token.json")
	os.Remove("profile.json")
	
	t.isLoggedIn = false
	t.userProfile = nil
	t.playlists = nil
	t.currentTracks = nil
	
	t.setupSidebar()
	t.mainContent.Clear()
	
	logoutText := tview.NewTextView().
		SetText("✅ Successfully logged out!\n\nAll login data has been cleared.\nPress 'l' to login again.").
		SetTextColor(successColor).
		SetBorder(true).
		SetTitle(" Logout ").
		SetBorderColor(borderColor)
	
	t.mainContent.AddItem(logoutText, 0, 1, false)
	t.updateStatusBar()
}

func (t *TUIApp) refreshData() {
	if !t.isLoggedIn {
		return
	}
	
	// Refresh login status and profile
	t.checkLoginStatus()
	t.setupSidebar()
	
	// If we're currently viewing playlists, refresh them
	if t.currentPanel == "playlists" || t.currentPanel == "tracks" {
		t.showPlaylistPanel()
	}
	
	t.updateStatusBar()
}

func (t *TUIApp) showProfilePanel() {
	if !t.isLoggedIn || t.userProfile == nil {
		return
	}
	
	// Clear main content
	t.mainContent.Clear()
	
	t.profileView = tview.NewTextView()
	profileText := fmt.Sprintf(`👤 User Profile

Name: %s
Email: %s
Spotify ID: %s
Spotify URL: %s

Press Esc to go back to navigation.`,
		t.userProfile.Username,
		t.userProfile.Email,
		t.userProfile.Userid,
		t.userProfile.ExternalURLs.Spotify)
	
	t.profileView.SetText(profileText).
		SetTextColor(tcell.ColorWhite).
		SetBorder(true).
		SetTitle(" Profile ").
		SetBorderColor(borderColor)
	
	t.mainContent.AddItem(t.profileView, 0, 1, false)
	t.currentPanel = "profile"
	t.updateStatusBar()
}

func (t *TUIApp) showPlaylistPanel() {
	if !t.isLoggedIn {
		return
	}
	
	// Clear main content
	t.mainContent.Clear()
	
	// Create playlist list
	t.playlistList = tview.NewList().
		SetMainTextColor(tcell.ColorWhite).
		SetSelectedTextColor(selectedColor).
		SetSelectedBackgroundColor(primaryColor)
	
	t.playlistList.SetBorder(true).
		SetTitle(" Playlists ").
		SetBorderColor(borderColor)
	
	// Load playlists
	t.loadPlaylists()
	
	// Create tracks table
	t.trackList = tview.NewTable().
		SetBorders(false).
		SetSeparator('│').
		SetSelectable(true, false).
		SetFixed(1, 0) // Fix header row
	
	t.trackList.SetBorder(true).
		SetTitle(" Tracks ").
		SetBorderColor(borderColor)
	
	// Layout: playlist list on left, tracks on right
	contentFlex := tview.NewFlex().
		AddItem(t.playlistList, 0, 1, true).
		AddItem(t.trackList, 0, 2, false)
	
	t.mainContent.AddItem(contentFlex, 0, 1, true)
	
	t.currentPanel = "playlists"
	t.app.SetFocus(t.playlistList)
	t.updateStatusBar()
}

func (t *TUIApp) showSearchPanel() {
	if !t.isLoggedIn {
		return
	}
	
	// Clear main content
	t.mainContent.Clear()
	
	// Create simple search input field
	t.searchInput = tview.NewInputField().
		SetLabel("Search: ").
		SetFieldBackgroundColor(tcell.ColorBlack).
		SetFieldTextColor(tcell.ColorWhite)
	
	t.searchInput.SetBorder(true).
		SetTitle(" Search Songs ")
	
	// Create search results table
	t.searchResults = tview.NewTable().
		SetBorders(false).
		SetSeparator('│').
		SetSelectable(true, false)
	
	t.searchResults.SetBorder(true).
		SetTitle(" Search Results ").
		SetBorderColor(borderColor)
	
	// Set up input field behavior
	t.searchInput.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			query := t.searchInput.GetText()
			if query != "" {
				t.performSearch(query)
			}
		}
	}).SetChangedFunc(func(text string) {
		// Optional: Perform live search as user types
		if len(text) >= 3 {
			t.performSearch(text)
		}
	})
	
	// Layout: search input on top, results below
	contentFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(t.searchInput, 3, 0, true).
		AddItem(t.searchResults, 0, 1, false)
	
	t.mainContent.AddItem(contentFlex, 0, 1, true)
	
	t.currentPanel = "search"
	t.app.SetFocus(t.searchInput)
	t.updateStatusBar()
}

func (t *TUIApp) loadPlaylists() {
	if !t.isLoggedIn || t.userProfile == nil {
		return
	}
	
	t.playlistList.Clear()
	t.playlistList.AddItem("Loading playlists...", "", 0, nil)
	
	// Load playlists in background
	go func() {
		client, err := utils.NewSpotifyClient()
		if err != nil {
			t.app.QueueUpdateDraw(func() {
				t.playlistList.Clear()
				t.playlistList.AddItem("❌ Error loading playlists", err.Error(), 0, nil)
			})
			return
		}
		
		url := "https://api.spotify.com/v1/users/" + t.userProfile.Userid + "/playlists"
		resp, err := client.Get(url)
		if err != nil {
			t.app.QueueUpdateDraw(func() {
				t.playlistList.Clear()
				t.playlistList.AddItem("❌ HTTP Error", err.Error(), 0, nil)
			})
			return
		}
		defer resp.Body.Close()
		
		var playlistsResp PlaylistsResponse
		err = json.NewDecoder(resp.Body).Decode(&playlistsResp)
		if err != nil {
			t.app.QueueUpdateDraw(func() {
				t.playlistList.Clear()
				t.playlistList.AddItem("❌ Parse Error", err.Error(), 0, nil)
			})
			return
		}
		
		t.playlists = playlistsResp.Items
		
		// Update UI on main thread
		t.app.QueueUpdateDraw(func() {
			t.playlistList.Clear()
			
			if len(t.playlists) == 0 {
				t.playlistList.AddItem("No playlists found", "", 0, nil)
				return
			}
			
			for i, playlist := range t.playlists {
				index := i // Capture loop variable
				t.playlistList.AddItem(
					fmt.Sprintf("🎵 %s", playlist.Name),
					fmt.Sprintf("%s tracks", ""),
					rune('1'+i%9),
					func() { t.loadPlaylistTracks(index) },
				)
			}
		})
	}()
}

func (t *TUIApp) performSearch(query string) {
	// Clear and show loading
	t.searchResults.Clear()
	t.searchResults.SetCell(0, 0, tview.NewTableCell("Searching...").SetTextColor(tcell.ColorYellow))
	
	// Perform search in background
	go func() {
		client, err := utils.NewSpotifyClient()
		if err != nil {
			t.app.QueueUpdateDraw(func() {
				t.searchResults.Clear()
				t.searchResults.SetCell(0, 0, tview.NewTableCell("❌ Error: "+err.Error()).SetTextColor(errorColor))
			})
			return
		}
		
		baseURL, err := url.Parse("https://api.spotify.com/v1/search")
		if err != nil {
			t.app.QueueUpdateDraw(func() {
				t.searchResults.Clear()
				t.searchResults.SetCell(0, 0, tview.NewTableCell("❌ Invalid URL").SetTextColor(errorColor))
			})
			return
		}
		
		params := url.Values{}
		params.Add("q", query)
		params.Add("type", "track")
		params.Add("limit", "20") // Increased limit for TUI
		baseURL.RawQuery = params.Encode()
		
		resp, err := client.Get(baseURL.String())
		if err != nil {
			t.app.QueueUpdateDraw(func() {
				t.searchResults.Clear()
				t.searchResults.SetCell(0, 0, tview.NewTableCell("❌ HTTP Error: "+err.Error()).SetTextColor(errorColor))
			})
			return
		}
		defer resp.Body.Close()
		
		var result SearchResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.app.QueueUpdateDraw(func() {
				t.searchResults.Clear()
				t.searchResults.SetCell(0, 0, tview.NewTableCell("❌ Parse Error: "+err.Error()).SetTextColor(errorColor))
			})
			return
		}
		
		t.searchTracks = result.Tracks.Items
		
		// Update UI on main thread
		t.app.QueueUpdateDraw(func() {
			t.searchResults.Clear()
			
			if len(t.searchTracks) == 0 {
				t.searchResults.SetCell(0, 0, tview.NewTableCell("No results found for '"+query+"'").SetTextColor(tcell.ColorGray))
				return
			}
			
			// Add header
			t.searchResults.SetCell(0, 0, tview.NewTableCell("#").SetTextColor(selectedColor).SetSelectable(false))
			t.searchResults.SetCell(0, 1, tview.NewTableCell("Track").SetTextColor(selectedColor).SetSelectable(false))
			t.searchResults.SetCell(0, 2, tview.NewTableCell("Artist").SetTextColor(selectedColor).SetSelectable(false))
			
			// Add search results
			for i, track := range t.searchTracks {
				row := i + 1
				
				t.searchResults.SetCell(row, 0, 
					tview.NewTableCell(fmt.Sprintf("%d", i+1)).SetTextColor(tcell.ColorGray))
				
				t.searchResults.SetCell(row, 1, 
					tview.NewTableCell(track.Name).SetTextColor(tcell.ColorWhite))
				
				artists := ""
				for j, artist := range track.Artists {
					if j > 0 {
						artists += ", "
					}
					artists += artist.Name
				}
				t.searchResults.SetCell(row, 2, 
					tview.NewTableCell(artists).SetTextColor(tcell.ColorLightBlue))
			}
			
			// Update title with result count
			t.searchResults.SetTitle(fmt.Sprintf(" Search Results (%d tracks) ", len(t.searchTracks)))
			
			// Auto-focus on search results after search
			if len(t.searchTracks) > 0 {
				t.currentPanel = "search_results"
				t.app.SetFocus(t.searchResults)
				t.updateStatusBar()
			}
		})
	}()
}

func (t *TUIApp) loadPlaylistTracks(playlistIndex int) {
	if playlistIndex >= len(t.playlists) {
		return
	}
	
	playlist := t.playlists[playlistIndex]
	
	// Clear and show loading
	t.trackList.Clear()
	t.trackList.SetCell(0, 0, tview.NewTableCell("Loading tracks...").SetTextColor(tcell.ColorYellow))
	
	// Load tracks in background
	go func() {
		client, err := utils.NewSpotifyClient()
		if err != nil {
			t.app.QueueUpdateDraw(func() {
				t.trackList.Clear()
				t.trackList.SetCell(0, 0, tview.NewTableCell("❌ Error: "+err.Error()).SetTextColor(errorColor))
			})
			return
		}
		
		var allTracks []PlaylistTrack
		next := playlist.Tracks.Href
		
		for next != "" {
			resp, err := client.Get(next)
			if err != nil {
				t.app.QueueUpdateDraw(func() {
					t.trackList.Clear()
					t.trackList.SetCell(0, 0, tview.NewTableCell("❌ HTTP Error: "+err.Error()).SetTextColor(errorColor))
				})
				return
			}
			
			var tracksResp PlaylistTracksResponse
			err = json.NewDecoder(resp.Body).Decode(&tracksResp)
			resp.Body.Close()
			
			if err != nil {
				t.app.QueueUpdateDraw(func() {
					t.trackList.Clear()
					t.trackList.SetCell(0, 0, tview.NewTableCell("❌ Parse Error: "+err.Error()).SetTextColor(errorColor))
				})
				return
			}
			
			allTracks = append(allTracks, tracksResp.Items...)
			next = tracksResp.Next
		}
		
		t.currentTracks = allTracks
		t.currentPlaylist = playlist.Name
		
		// Update UI on main thread
		t.app.QueueUpdateDraw(func() {
			// Reset pagination and calculate pages
			t.currentPage = 0
			t.calculatePagination()
			
			// Display first page
			t.displayCurrentTracksPage()
			
			// Update title with track count and pagination info
			if t.totalPages > 1 {
				t.trackList.SetTitle(fmt.Sprintf(" %s (%d tracks, %d pages) ", playlist.Name, len(allTracks), t.totalPages))
			} else {
				t.trackList.SetTitle(fmt.Sprintf(" %s (%d tracks) ", playlist.Name, len(allTracks)))
			}
			
			// Auto-switch focus to tracks panel when tracks are loaded
			if len(allTracks) > 0 {
				t.currentPanel = "tracks"
				t.app.SetFocus(t.trackList)
			}
			
			// Update status bar to show pagination controls
			t.updateStatusBar()
		})
	}()
}

// Pagination methods
func (t *TUIApp) calculatePagination() {
	if len(t.currentTracks) == 0 {
		t.totalPages = 0
		t.currentPage = 0
		return
	}
	
	t.totalPages = (len(t.currentTracks) + t.tracksPerPage - 1) / t.tracksPerPage
	if t.currentPage >= t.totalPages {
		t.currentPage = t.totalPages - 1
	}
	if t.currentPage < 0 {
		t.currentPage = 0
	}
}

func (t *TUIApp) nextPage() {
	if t.currentPage < t.totalPages-1 {
		t.currentPage++
		t.displayCurrentTracksPage()
		t.updateStatusBar()
		// Update title to reflect current page
		if t.currentPlaylist != "" && t.totalPages > 1 {
			t.trackList.SetTitle(fmt.Sprintf(" %s (Page %d/%d) ", t.currentPlaylist, t.currentPage+1, t.totalPages))
		}
	}
}

func (t *TUIApp) previousPage() {
	if t.currentPage > 0 {
		t.currentPage--
		t.displayCurrentTracksPage()
		t.updateStatusBar()
		// Update title to reflect current page
		if t.currentPlaylist != "" && t.totalPages > 1 {
			t.trackList.SetTitle(fmt.Sprintf(" %s (Page %d/%d) ", t.currentPlaylist, t.currentPage+1, t.totalPages))
		}
	}
}

func (t *TUIApp) firstPage() {
	if t.totalPages > 0 {
		t.currentPage = 0
		t.displayCurrentTracksPage()
		t.updateStatusBar()
		// Update title to reflect current page
		if t.currentPlaylist != "" && t.totalPages > 1 {
			t.trackList.SetTitle(fmt.Sprintf(" %s (Page %d/%d) ", t.currentPlaylist, t.currentPage+1, t.totalPages))
		}
	}
}

func (t *TUIApp) lastPage() {
	if t.totalPages > 0 {
		t.currentPage = t.totalPages - 1
		t.displayCurrentTracksPage()
		t.updateStatusBar()
		// Update title to reflect current page
		if t.currentPlaylist != "" && t.totalPages > 1 {
			t.trackList.SetTitle(fmt.Sprintf(" %s (Page %d/%d) ", t.currentPlaylist, t.currentPage+1, t.totalPages))
		}
	}
}

func (t *TUIApp) displayCurrentTracksPage() {
	t.trackList.Clear()
	
	if len(t.currentTracks) == 0 {
		t.trackList.SetCell(0, 0, tview.NewTableCell("No tracks found").SetTextColor(tcell.ColorGray))
		return
	}
	
	// Ensure table is selectable
	t.trackList.SetSelectable(true, false)
	
	// Add header
	t.trackList.SetCell(0, 0, tview.NewTableCell("#").SetTextColor(selectedColor).SetSelectable(false))
	t.trackList.SetCell(0, 1, tview.NewTableCell("Track").SetTextColor(selectedColor).SetSelectable(false))
	t.trackList.SetCell(0, 2, tview.NewTableCell("Artist").SetTextColor(selectedColor).SetSelectable(false))
	t.trackList.SetCell(0, 3, tview.NewTableCell("Album").SetTextColor(selectedColor).SetSelectable(false))
	
	// Calculate start and end indices for current page
	start := t.currentPage * t.tracksPerPage
	end := start + t.tracksPerPage
	if end > len(t.currentTracks) {
		end = len(t.currentTracks)
	}
	
	// Add tracks for current page
	for i := start; i < end; i++ {
		item := t.currentTracks[i]
		row := i - start + 1 // Row in table (1-indexed, accounting for header)
		
		t.trackList.SetCell(row, 0, 
			tview.NewTableCell(fmt.Sprintf("%d", i+1)).SetTextColor(tcell.ColorGray))
		
		t.trackList.SetCell(row, 1, 
			tview.NewTableCell(item.Track.Name).SetTextColor(tcell.ColorWhite))
		
		artists := ""
		for j, artist := range item.Track.Artists {
			if j > 0 {
				artists += ", "
			}
			artists += artist.Name
		}
		t.trackList.SetCell(row, 2, 
			tview.NewTableCell(artists).SetTextColor(tcell.ColorLightBlue))
		
		// Note: Album info would need to be added to Track struct if needed
		// t.trackList.SetCell(row, 3, 
		//     tview.NewTableCell(item.Track.Album.Name).SetTextColor(tcell.ColorGreen))
	}
	
	// Set selection to first track if we have tracks
	if end > start {
		t.trackList.Select(1, 0) // Select first track (row 1, after header)
	}
}

// Playback methods
func (t *TUIApp) playSelectedTrack() {
	if len(t.currentTracks) == 0 {
		return
	}
	
	row, _ := t.trackList.GetSelection()
	if row <= 0 { // Header row or no selection
		return
	}
	
	// Calculate actual track index based on current page and selection
	start := t.currentPage * t.tracksPerPage
	trackIndex := start + row - 1 // -1 because row 0 is header
	
	if trackIndex >= len(t.currentTracks) {
		return
	}
	
	track := t.currentTracks[trackIndex]
	if track.Track.URI == "" {
		t.statusBar.SetText(" Error: Track URI not available ")
		return
	}
	
	// Use playlist context if available, otherwise individual track
	if t.currentPlaylist != "" {
		// Find the playlist URI
		var playlistURI string
		for _, playlist := range t.playlists {
			if playlist.Name == t.currentPlaylist {
				playlistURI = playlist.Uri
				break
			}
		}
		
		if playlistURI != "" {
			// Play from playlist context with offset to selected track
			go func() {
				StartMusic(&playlistURI, nil)
				t.app.QueueUpdateDraw(func() {
					t.isPlaying = true
					t.currentTrackURI = track.Track.URI
					// Update status will show playing indicator
					t.updateStatusBar()
				})
			}()
		} else {
			// Fallback to individual track
			t.playIndividualTrack(track.Track)
		}
	} else {
		// Play individual track
		t.playIndividualTrack(track.Track)
	}
}

func (t *TUIApp) playSelectedSearchTrack() {
	if len(t.searchTracks) == 0 {
		return
	}
	
	row, _ := t.searchResults.GetSelection()
	if row <= 0 { // Header row or no selection
		return
	}
	
	trackIndex := row - 1 // -1 because row 0 is header
	if trackIndex >= len(t.searchTracks) {
		return
	}
	
	track := t.searchTracks[trackIndex]
	if track.URI == "" {
		t.statusBar.SetText(" Error: Track URI not available ")
		return
	}
	
	// Play individual track from search results
	uris := []string{track.URI}
	go func() {
		StartMusic(nil, &uris)
		t.app.QueueUpdateDraw(func() {
			t.isPlaying = true
			t.currentTrackURI = track.URI
			// Update status will show playing indicator
			t.updateStatusBar()
		})
	}()
}

func (t *TUIApp) playIndividualTrack(track Track) {
	uris := []string{track.URI}
	go func() {
		StartMusic(nil, &uris)
		t.app.QueueUpdateDraw(func() {
			t.isPlaying = true
			t.currentTrackURI = track.URI
			// Update status will show playing indicator
			t.updateStatusBar()
		})
	}()
}

func (t *TUIApp) togglePlayback() {
	if t.isPlaying {
		go func() {
			PausePlayback()
			t.app.QueueUpdateDraw(func() {
				t.isPlaying = false
				// Status bar will show paused state
				t.updateStatusBar()
			})
		}()
	} else {
		go func() {
			ResumePlayback()
			t.app.QueueUpdateDraw(func() {
				t.isPlaying = true
				// Status bar will show playing state
				t.updateStatusBar()
			})
		}()
	}
}

func (t *TUIApp) getArtistNames(artists []Artist) string {
	var names []string
	for _, artist := range artists {
		names = append(names, artist.Name)
	}
	return strings.Join(names, ", ")
}

func (t *TUIApp) getSearchArtistNames(artists []ArtistResp) string {
	var names []string
	for _, artist := range artists {
		names = append(names, artist.Name)
	}
	return strings.Join(names, ", ")
}

func (t *TUIApp) showMessage(message string, color tcell.Color) {
	// Just update status bar to avoid any UI corruption
	t.statusBar.SetText(fmt.Sprintf(" %s ", message))
	t.statusBar.SetBackgroundColor(color)
}

// Playback wrapper methods for sidebar
func (t *TUIApp) pausePlayback() {
	PausePlayback()
	t.app.QueueUpdateDraw(func() {
		t.isPlaying = false
		t.updateStatusBar()
	})
}

func (t *TUIApp) resumePlayback() {
	ResumePlayback()
	t.app.QueueUpdateDraw(func() {
		t.isPlaying = true
		t.updateStatusBar()
	})
}

func (t *TUIApp) nextTrack() {
	NextTrack()
}

func (t *TUIApp) previousTrack() {
	PreviousTrack()
}

// TUI command
var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch the Terminal User Interface",
	Long:  "Launch GitifyTUI - A beautiful terminal interface for Spotify, inspired by Lazygit",
	Run: func(cmd *cobra.Command, args []string) {
		app := NewTUIApp()
		if err := app.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	spotifyCmd.AddCommand(tuiCmd)
}