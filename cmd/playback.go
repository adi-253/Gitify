// this will be a helper function rather than a cli command because we will integrate with search and playlist
package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/adi-253/Gitify/cmd/utils"
	"github.com/spf13/cobra"
)

type PlaybackRequest struct {
    ContextURI *string        `json:"context_uri,omitempty"`
    Uris       *[]string      `json:"uris,omitempty"`
    Offset     *PlaybackOffset `json:"offset,omitempty"`
    PositionMS *int           `json:"position_ms,omitempty"`
}

type PlaybackOffset struct {
    Position *int    `json:"position,omitempty"`
    URI      *string `json:"uri,omitempty"`
}

// CurrentPlayback represents the response from Spotify's current playback endpoint
type CurrentPlayback struct {
    IsPlaying bool `json:"is_playing"`
    Item      *struct {
        Name    string `json:"name"`
        URI     string `json:"uri"`
        Artists []struct {
            Name string `json:"name"`
        } `json:"artists"`
    } `json:"item"`
}

// PlaybackInfo holds simplified playback information for the TUI
type PlaybackInfo struct {
    IsPlaying  bool
    TrackName  string
    ArtistName string
    TrackURI   string
}

// When Gitify is running inside the Bubble Tea TUI we don't want the
// playback helpers to print directly to stdout because that corrupts
// the screen layout and makes list highlighting look wrong.
var silentPlayback bool

// SetPlaybackSilent controls whether playback helpers print messages.
// The TUI sets this to true; CLI commands keep the default (false).
func SetPlaybackSilent(s bool) {
	silentPlayback = s
}

// GetCurrentPlayback fetches the current playback state from Spotify
func GetCurrentPlayback() (*PlaybackInfo, error) {
    client, err := utils.NewSpotifyClient()
    if err != nil {
        return nil, err
    }

    resp, err := client.Get("https://api.spotify.com/v1/me/player/currently-playing")
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    // 204 means no content (nothing playing)
    if resp.StatusCode == 204 {
        return &PlaybackInfo{IsPlaying: false}, nil
    }

    if resp.StatusCode != 200 {
        return nil, fmt.Errorf("failed to get playback: status %d", resp.StatusCode)
    }

    var playback CurrentPlayback
    if err := json.NewDecoder(resp.Body).Decode(&playback); err != nil {
        return nil, err
    }

    info := &PlaybackInfo{
        IsPlaying: playback.IsPlaying,
    }

    if playback.Item != nil {
        info.TrackName = playback.Item.Name
        info.TrackURI = playback.Item.URI
        var artists []string
        for _, a := range playback.Item.Artists {
            artists = append(artists, a.Name)
        }
        if len(artists) > 0 {
            info.ArtistName = artists[0]
            if len(artists) > 1 {
                for i := 1; i < len(artists); i++ {
                    info.ArtistName += ", " + artists[i]
                }
            }
        }
    }

    return info, nil
}

func StartMusic(contextURI *string, uris *[]string) {
    StartMusicWithOffset(contextURI, uris, nil)
}

// StartMusicWithOffset starts playback with an optional offset position
func StartMusicWithOffset(contextURI *string, uris *[]string, offsetPosition *int) {
    req := PlaybackRequest{
        ContextURI: contextURI,
        Uris:       uris,
        Offset:     nil,
        PositionMS: nil,
    }

    // If offset position is provided, set it
    if offsetPosition != nil {
        req.Offset = &PlaybackOffset{
            Position: offsetPosition,
        }
    }

	
    buf := new(bytes.Buffer)
    json.NewEncoder(buf).Encode(req)

    client, err := utils.NewSpotifyClient()
    if err != nil {
        if !silentPlayback {
            fmt.Printf("Error creating Spotify client: %v\n", err)
        }
        return
    }
    
    resp, err := client.Put("https://api.spotify.com/v1/me/player/play", buf)
    if err != nil {
        if !silentPlayback {
            fmt.Printf("Error starting playback: %v\n", err)
        }
        return
    }
    defer resp.Body.Close()
    
    if resp.StatusCode == 404 {
        if !silentPlayback {
            fmt.Println("No active device found. Please:")
            fmt.Println("   1. Open Spotify app on your phone/computer")
            fmt.Println("   2. Start playing any song briefly")
            fmt.Println("   3. Then try again")
        }
        return
    }
    
    if resp.StatusCode != 204 {
        if !silentPlayback {
            fmt.Printf("Playback failed with status: %d\n", resp.StatusCode)
        }
        return
    }
    
    if !silentPlayback {
        fmt.Println("Playback started successfully!")
    }
}

func PausePlayback() {
    client, err := utils.NewSpotifyClient()
    if err != nil {
        if !silentPlayback {
            fmt.Printf("Error creating Spotify client: %v\n", err)
        }
        return
    }

    resp, err := client.Put("https://api.spotify.com/v1/me/player/pause", nil)
    if err != nil {
        if !silentPlayback {
            fmt.Printf("Error pausing playback: %v\n", err)
        }
        return
    }
    defer resp.Body.Close()

    if !silentPlayback {
        switch resp.StatusCode {
        case 200:
            fmt.Println("Playback paused")
        case 404:
            fmt.Println("No active device found")
        case 403:
            fmt.Println("Playback control requires Spotify Premium")
        default:
            fmt.Printf("Failed to pause with status: %d\n", resp.StatusCode)
        }
    }
}

func ResumePlayback() {
    client, err := utils.NewSpotifyClient()
    if err != nil {
        if !silentPlayback {
            fmt.Printf("Error creating Spotify client: %v\n", err)
        }
        return
    }

    resp, err := client.Put("https://api.spotify.com/v1/me/player/play", nil)
    if err != nil {
        if !silentPlayback {
            fmt.Printf("Error resuming playback: %v\n", err)
        }
        return
    }
    defer resp.Body.Close()

    if !silentPlayback {
        switch resp.StatusCode {
        case 200:
            fmt.Println("Playback resumed")
        case 404:
            fmt.Println("No active device found")
        case 403:
            fmt.Println("Playback control requires Spotify Premium")
        default:
            fmt.Printf("Failed to resume with status: %d\n", resp.StatusCode)
        }
    }
}

func NextTrack() {
    client, err := utils.NewSpotifyClient()
    if err != nil {
        if !silentPlayback {
            fmt.Printf("Error creating Spotify client: %v\n", err)
        }
        return
    }

    resp, err := client.Post("https://api.spotify.com/v1/me/player/next", nil)
    if err != nil {
        if !silentPlayback {
            fmt.Printf("Error skipping track: %v\n", err)
        }
        return
    }
    defer resp.Body.Close()

    if !silentPlayback {
        switch resp.StatusCode {
        case 200:
            fmt.Println("Skipped to next track")
        case 404:
            fmt.Println("No active device found")
        case 403:
            fmt.Println("Playback control requires Spotify Premium")
        default:
            fmt.Printf("Failed to skip with status: %d\n", resp.StatusCode)
        }
    }
}

func PreviousTrack() {
    client, err := utils.NewSpotifyClient()
    if err != nil {
        if !silentPlayback {
            fmt.Printf("Error creating Spotify client: %v\n", err)
        }
        return
    }

    resp, err := client.Post("https://api.spotify.com/v1/me/player/previous", nil)
    if err != nil {
        if !silentPlayback {
            fmt.Printf("Error going to previous track: %v\n", err)
        }
        return
    }
    defer resp.Body.Close()

    if !silentPlayback {
        switch resp.StatusCode {
        case 200:
            fmt.Println("Previous track")
        case 404:
            fmt.Println("No active device found")
        case 403:
            fmt.Println("Playback control requires Spotify Premium")
        default:
            fmt.Printf("Failed to go to previous track with status: %d\n", resp.StatusCode)
        }
    }
}

// CLI Commands
var pauseCmd = &cobra.Command{
	Use:   "pause",
	Short: "Pause current playback",
	Run: func(cmd *cobra.Command, args []string) {
		PausePlayback()
	},
}

var resumeCmd = &cobra.Command{
	Use:   "resume",
	Short: "Resume current playback", 
	Run: func(cmd *cobra.Command, args []string) {
		ResumePlayback()
	},
}

var nextCmd = &cobra.Command{
	Use:   "next",
	Short: "Skip to next track",
	Run: func(cmd *cobra.Command, args []string) {
		NextTrack()
	},
}

var prevCmd = &cobra.Command{
	Use:   "prev",
	Short: "Go to previous track",
	Run: func(cmd *cobra.Command, args []string) {
		PreviousTrack()
	},
}

func init() {
	spotifyCmd.AddCommand(pauseCmd)
	spotifyCmd.AddCommand(resumeCmd)
	spotifyCmd.AddCommand(nextCmd)
	spotifyCmd.AddCommand(prevCmd)
}
