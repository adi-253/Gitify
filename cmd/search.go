package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/adi-253/Gitify/cmd/utils"
	"github.com/spf13/cobra"
)

// Minimal, readable struct — only what you need
type SearchResponse struct {
	Tracks struct {
		Items []TrackItem `json:"items"`
	} `json:"tracks"`
}

type TrackItem struct {
	Name         string   `json:"name"`
	Artists      []ArtistResp `json:"artists"`
	ExternalURLs struct {
		Spotify string `json:"spotify"`
	} `json:"external_urls"`
	ID string `json:"id"` // Used for future play implementation
}

type ArtistResp struct {
	Name string `json:"name"`
}

var searchcmd = &cobra.Command{
	Use:   "search [song]",
	Short: "Search for a song on Spotify",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Println("Type a song name to play")
			return
		}

		song := strings.Join(args, " ")

		baseURL, err := url.Parse("https://api.spotify.com/v1/search")
		if err != nil {
			fmt.Println("Invalid base URL")
			return
		}

		params := url.Values{}
		params.Add("q", song)
		params.Add("type", "track")
		params.Add("limit", "10")  // since the limit is 10 and each req returns 10 no need pagination

		baseURL.RawQuery = params.Encode()

		client, err := utils.NewSpotifyClient()
		if err != nil {
			fmt.Println("Relogin and try again")
			return
		}

		resp, err := client.Get(baseURL.String())
		if err != nil {
			fmt.Println("Error fetching data:", err)
			return
		}
		defer resp.Body.Close()

		var result SearchResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			fmt.Println("Error parsing response:", err)
			return
		}

		if len(result.Tracks.Items) == 0 {
			fmt.Printf("No results found for '%s'\n", song)
			return
		}

		fmt.Printf("\nSearch Results for: %s\n", song)
		fmt.Println(strings.Repeat("=", 40))

		for i, track := range result.Tracks.Items {
			artistNames := make([]string, len(track.Artists))
			for j, artist := range track.Artists {
				artistNames[j] = artist.Name
			}

			fmt.Printf("%d. %s — %s\n", i+1, track.Name, strings.Join(artistNames, ", "))
			fmt.Printf("   %s\n\n", track.ExternalURLs.Spotify)
		}
	},
}

func init() {
	spotifyCmd.AddCommand(searchcmd)
}
