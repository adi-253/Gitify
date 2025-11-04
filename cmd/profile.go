package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"github.com/adi-253/Gitify/cmd/utils"
	"github.com/spf13/cobra"
	"net/http"
)

type Profile struct {
	Username string `json:"display_name"`
	Email string `json:"email"`
	ExternalURLs struct {
		Spotify string `json:"spotify"`
	} `json:"external_urls"`
	Userid string `json:"id"`

	
}



func getUserInfo() error {
	var ProfileData Profile

	url:= "https://api.spotify.com/v1/me"
	client,err:=utils.NewSpotifyClient()

	if err!=nil{
		return err
	}

	resp,err:=client.Get(url)
	
	if err!=nil{
		return err
	}
	
	if resp.StatusCode == http.StatusUnauthorized {
    	return fmt.Errorf("access token invalid or expired, please login again")
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&ProfileData); err != nil { // reason using this is because io.readall will read everything into memory first and then do the struct here directly what is needed is added from struct val
		return err
	}
	
	file,err:=os.Create("profile.json")
	
	if err!=nil{
		return fmt.Errorf("could not create json file for profile. Login again using login cmd")
	}
	
	defer file.Close()

	encoder:=json.NewEncoder(file)
	encoder.SetIndent(""," ")
	
	if err:=encoder.Encode(ProfileData);err!=nil{
		return fmt.Errorf("Could not write the user info to the file")
	}

	return nil
}


var profileCmd = &cobra.Command{
	Use: "me",
	Short: "fetch spotfiy user details",
	RunE: func(cmd *cobra.Command, args []string) error{
		var ProfileData Profile
		
		data, err := os.ReadFile("profile.json")
		
		if err != nil {
		return fmt.Errorf("Profile not found please login again")
		}
		
		if err:=json.Unmarshal(data,&ProfileData);err!=nil{
			return fmt.Errorf("Couldnot fetch profile data from the json")
		}
		
		fmt.Println("Username",ProfileData.Username)
		fmt.Println("Email",ProfileData.Email)
		fmt.Println("Link", ProfileData.ExternalURLs.Spotify)
		
		return nil
	},
}

func init(){
	spotfiyCmd.AddCommand(profileCmd)
}