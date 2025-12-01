package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/adi-253/Gitify/cmd/utils"
	"github.com/spf13/cobra"
)

// ---------------- Structs ----------------

// Only keep essential fields for CLI & playback integration
type PlaylistsResponse struct {
	Items []Playlist `json:"items"`
	Next  string     `json:"next"`
}

type Playlist struct {
	Name   string `json:"name"`
	ID     string `json:"id"`
	Tracks struct {
		Href string `json:"href"`
	} `json:"tracks"`
	Uri string `json:"uri"`
}

type PlaylistTracksResponse struct {
	Items []PlaylistTrack `json:"items"`
	Next  string           `json:"next"`
}

type PlaylistTrack struct {
	Track Track `json:"track"`
}

type Track struct {
	Name    string   `json:"name"`
	ID      string   `json:"id"`
	Artists []Artist `json:"artists"`
}

type Artist struct {
	Name string `json:"name"`
}

// ---------------- Command ----------------

var playlistCmd = &cobra.Command{
	Use:   "playlist",
	Short: "Fetch and view user playlists and tracks",
	Run: func(cmd *cobra.Command, args []string) {
		data, err := os.ReadFile("profile.json")
		if err != nil {
			fmt.Println("Could not get user data. Please login again.")
			return
		}

		var userinfo Profile
		err = json.Unmarshal(data, &userinfo)
		if err != nil {
			fmt.Println("Could not parse user data.")
			return
		}

		client, err := utils.NewSpotifyClient()
		if err != nil {
			fmt.Printf("Error creating Spotify client: %s\n", err)
			return
		}

		baseURL := "https://api.spotify.com/v1/users/" + userinfo.Userid + "/playlists"
		allPlaylists, err := fetchAllPlaylists(client, baseURL)
		if err != nil {
			fmt.Printf("Error fetching playlists: %s\n", err)
			return
		}

		if len(allPlaylists) == 0 {
			fmt.Println("No playlists found.")
			return
		}

		fmt.Printf("\n🎵 You have %d playlists:\n\n", len(allPlaylists))
		for i, p := range allPlaylists {
			fmt.Printf("[%d] %s\n", i+1, p.Name)
		}

		fmt.Print("\nEnter playlist number to view songs: ")
		var choice int
		_, err = fmt.Scan(&choice)
		if err != nil || choice < 1 || choice > len(allPlaylists) {
			fmt.Println("Invalid choice.")
			return
		}

		selected := allPlaylists[choice-1]
		fmt.Printf("\nFetching songs for: %s\n\n", selected.Name)

		tracks, err := fetchAllTracks(client, selected.Tracks.Href)
		if err != nil {
			fmt.Printf("Error fetching tracks: %s\n", err)
			return
		}

		for i, t := range tracks {
			fmt.Printf("%d. %s — %s\n", i+1, t.Name, joinArtists(t.Artists))
		}

		// Add playback options
		fmt.Print("\n🎵 Play Options:\n")
		fmt.Println("[P] Play entire playlist")
		fmt.Println("[Q] Quit")
		fmt.Print("\nChoose an option: ")
		
		var playChoice string
		fmt.Scan(&playChoice)
		
		switch strings.ToUpper(playChoice) {
		case "P":
			// Play entire playlist using context URI (format: spotify:playlist:ID)
			playlistURI := selected.Uri
			if playlistURI == "" {
				playlistURI = "spotify:playlist:" + selected.ID
			}

			fmt.Printf("\n🎶 Playing playlist: %s\n", selected.Name)
			StartMusic(&playlistURI, nil)
		case "Q":
			fmt.Println("Goodbye! 👋")
		default:
			fmt.Println("Invalid option.")
		}
	},
}

// ---------------- Helper Functions ----------------

func fetchAllPlaylists(client *utils.SpotifyClient, href string) ([]Playlist, error) {
	var all []Playlist
	next := href

	for next != "" {
		u, _ := url.Parse(next)
		q := u.Query()
		if q.Get("limit") == "" {
			q.Set("limit", "50")
		}
		u.RawQuery = q.Encode()

		resp, err := client.Get(u.String())
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		var res PlaylistsResponse
		if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
			return nil, err
		}

		all = append(all, res.Items...)
		next = res.Next
	}

	return all, nil
}

func fetchAllTracks(client *utils.SpotifyClient, href string) ([]Track, error) {
	var all []Track
	next := href

	for next != "" {
		u, _ := url.Parse(next)
		q := u.Query()
		if q.Get("limit") == "" {
			q.Set("limit", "100")
		}
		u.RawQuery = q.Encode()

		resp, err := client.Get(u.String())
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		var res PlaylistTracksResponse
		if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
			return nil, err
		}

		for _, item := range res.Items {
			all = append(all, item.Track)
		}

		next = res.Next
	}

	return all, nil
}

func joinArtists(artists []Artist) string {
	names := ""
	for i, a := range artists {
		if i > 0 {
			names += ", "
		}
		names += a.Name
	}
	return names
}

func init() {
	spotifyCmd.AddCommand(playlistCmd)
}
