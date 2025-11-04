package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"

	"github.com/adi-253/Gitify/cmd/utils"
	"github.com/spf13/cobra"
)


type PlaylistsResponse struct {
	Total int        `json:"total"`
	Items []Playlist `json:"items"`
}

type Playlist struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	ID          string      `json:"id"`
	Href        string      `json:"href"`
	Tracks      PlaylistRef `json:"tracks"`
}

type PlaylistRef struct {
	Href string `json:"href"`
}

type PlaylistTracksResponse struct {
	Href     string          `json:"href"`
	Limit    int             `json:"limit"`
	Next     string          `json:"next"`
	Offset   int             `json:"offset"`
	Total    int             `json:"total"`
	Items    []PlaylistTrack `json:"items"`
}

type PlaylistTrack struct {
	Track Track `json:"track"`
}

type Track struct {
	Name    string   `json:"name"`
	ID      string   `json:"id"`
	Href    string   `json:"href"`
	Artists []Artist `json:"artists"`
	Album   Album    `json:"album"`
}

type Artist struct {
	Name string `json:"name"`
}

type Album struct {
	Name string `json:"name"`
}

// --------- Command ---------

var playlistCmd = &cobra.Command{
	Use:   "show playlist",
	Short: "Fetching Playlist Info",
	Run: func(c *cobra.Command, args []string) {
		data, err := os.ReadFile("profile.json")
		if err != nil {
			fmt.Println("Could not get user data. Login again")
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

		url := "https://api.spotify.com/v1/users/" + userinfo.Userid + "/playlists"
		resp, err := client.Get(url)
		if err != nil {
			fmt.Printf("HTTP request failed: %s\n", err)
			return
		}
		defer resp.Body.Close()

		var playlists PlaylistsResponse
		err = json.NewDecoder(resp.Body).Decode(&playlists)
		if err != nil {
			fmt.Printf("Error decoding JSON response: %s\n", err)
			return
		}

		if len(playlists.Items) == 0 {
			fmt.Println("No playlists found.")
			return
		}

		fmt.Printf("Total Playlists: %d\n\n", playlists.Total)
		for i, p := range playlists.Items {
			fmt.Printf("[%d] %s\n", i+1, p.Name)
		}

		fmt.Print("\nEnter playlist number: ")
		var choice int
		_, err = fmt.Scan(&choice)
		if err != nil || choice < 1 || choice > len(playlists.Items) {
			fmt.Println("Invalid choice.")
			return
		}

		selected := playlists.Items[choice-1]
		fmt.Printf("\nFetching tracks for: %s\n", selected.Name)

		fetchAndPrintTracks(client, selected.Tracks.Href)
	},
}

// --------- Helper for pagination ---------

func fetchAndPrintTracks(client *utils.SpotifyClient, href string) {
	next := href
	for next != "" {
		// Add limit and offset explicitly if missing
		u, _ := url.Parse(next)
		q := u.Query()
		if q.Get("limit") == "" {
			q.Set("limit", "50")
		}
		u.RawQuery = q.Encode()

		resp, err := client.Get(u.String())
		if err != nil {
			fmt.Printf("Failed to fetch tracks: %s\n", err)
			return
		}
		defer resp.Body.Close()

		var tracksResp PlaylistTracksResponse
		err = json.NewDecoder(resp.Body).Decode(&tracksResp)
		if err != nil {
			fmt.Printf("Error decoding tracks JSON: %s\n", err)
			return
		}

		for i, item := range tracksResp.Items {
			fmt.Printf("%d. %s — %s (%s)\n", i+1+tracksResp.Offset,
				item.Track.Name,
				joinArtists(item.Track.Artists),
				item.Track.Album.Name)
		}

		next = tracksResp.Next
	}
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


func init(){
	spotfiyCmd.AddCommand(playlistCmd)
}
