package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/adi-253/Gitify/cmd/utils"
	"github.com/spf13/cobra"
)
type Profile struct {
	Username string `json:"display_name"`
	Email string `json:"email"`
	ExternalURLs struct {
		Spotify string `json:"spotify"`
	} `json:"external_urls"`

	
}
var profileCmd = &cobra.Command{
	Use: "me",
	Short: "fetch spotfiy user details",
	RunE: func(cmd *cobra.Command, args []string) error{
			var Response Profile
			url:= "https://api.spotify.com/v1/me"
			client,err:=utils.NewSpotifyClient()
			if err!=nil{
				return err
			}
			resp,err:=client.Get(url)
			if err!=nil{
				return err
			}
			if err := json.NewDecoder(resp.Body).Decode(&Response); err != nil { // reason using this is because io.readall will read everything into memory first and then do the struct here directly what is needed is added from struct val
				return err
			}
			fmt.Println("Username",Response.Username)
			fmt.Println("Email",Response.Email)
			fmt.Println("Link",Response.ExternalURLs.Spotify)
			return nil
	},
}

func init(){
	spotfiyCmd.AddCommand(profileCmd)
}