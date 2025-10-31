// TODO - Create a UNIVERSAL HTTP CLIENT INSTEAD OF OPENING AND CLOSING AGAIN WHILE CALLING APIS

// use mux router only when multiple routes are used
// This is all logics of OAUTH 2.0 used in Spotify
package cmd

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/joho/godotenv"
)


var RedirectUrl string
var Client_ID string
var Client_Secret string

func init() { // this will be called because cmd folder is imported in main.go( you need to think in terms of main.go because thats where everythig is done)
	// Load .env file directly here to ensure it's loaded before we read env vars
	godotenv.Load()
	
	Client_ID = os.Getenv("CLIENT_ID")
	Client_Secret = os.Getenv("CLIENT_SECRET")
	RedirectUrl = os.Getenv("REDIRECT_URL")
	
	if RedirectUrl == "" {
		RedirectUrl = "http://localhost:8080/callback"
	}
}

type SpotfiyToken struct {
	AccessToken string `json:"access_token"`
	TokenType string    `json:"token_type"`
	RefreshToken string	 `json:"refresh_token"`
	Scope string		`json:"scope"`
	ExpiresIn int		`json:"expires_in"`
}


func generateRandomString(length int) (string, error) {
	bytes := make([]byte, length/2) // 2 hex chars per byte
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}


func exchangeToken(code string) (*SpotfiyToken, error) {
	tokenURL := "https://accounts.spotify.com/api/token" // here the request goes in encoded form and it is POST so it is not query params

	data:= url.Values{}  // this is used for form encoded data or query parameters (here it is form encoded)
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri",RedirectUrl)

	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(Client_ID,Client_Secret)

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token endpoint returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var token SpotfiyToken
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, err
	}

	return &token, nil
}


// callback function is used by spotify.com/authorize when that endpoint is hit by login
func HandleCallback(w http.ResponseWriter, r *http.Request){
	code := r.URL.Query().Get("code")
	if code==""{
		http.Error(w,"No code in request",http.StatusBadRequest)
		return 
	}

	token, err := exchangeToken(code)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to exchange token: %v", err), http.StatusInternalServerError)
		return
	}

	file, err := os.Create("token.json")
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to create token file: %v", err), http.StatusInternalServerError)
		return
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(token); err != nil {
		http.Error(w, fmt.Sprintf("failed to write token to file: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(" Login successful! You can close this tab now."))

	fmt.Println("Access token saved successfully to token.json")
}


func LoginHandler(w http.ResponseWriter, r *http.Request){
	state, err:=generateRandomString(16)
	if err!=nil{
		http.Error(w,"failed to generate random string",http.StatusBadRequest)
		return
	}
	
	scope := "user-read-private user-read-email user-library-read playlist-read-private"
	authURL, _ := url.Parse("https://accounts.spotify.com/authorize")
	params := url.Values{}
	params.Add("client_id", Client_ID)
	params.Add("response_type", "code")
	params.Add("redirect_uri", RedirectUrl)
	params.Add("scope", scope)
	params.Add("state", state)

	authURL.RawQuery = params.Encode()

	http.Redirect(w, r, authURL.String(), http.StatusFound)
}

func RefreshToken() error {
	var existing_token SpotfiyToken

	
	file, err := os.ReadFile("token.json")
	if err != nil {
		return err
	}

	err = json.Unmarshal(file, &existing_token)
	if err != nil {
		return err
	}

	tokenURL := "https://accounts.spotify.com/api/token"

	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", existing_token.RefreshToken)

	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(Client_ID,Client_Secret)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	
	body, err := io.ReadAll(resp.Body) // since you are using io.Readall() readall takes in all the incoming chunks of the data all togehter first and then converts to bytes so you cant use json.Encoder and decoder if you dont do this you can do the json.Encoder and Decoder
	if err != nil {
		return err
	}

	var new_token SpotfiyToken
	err = json.Unmarshal(body, &new_token)
	if err != nil {
		return err
	}

	existing_token.AccessToken = new_token.AccessToken
	existing_token.TokenType = new_token.TokenType
	existing_token.ExpiresIn = new_token.ExpiresIn

	// Refresh token may or may not be present
	if new_token.RefreshToken != "" {
		existing_token.RefreshToken = new_token.RefreshToken
	}

	updatedData, err := json.MarshalIndent(existing_token, "", "  ")
	if err != nil {
		return err
	}

	err = os.WriteFile("token.json", updatedData, 0644)
	if err != nil {
		return err
	}

	fmt.Println("Access token refreshed successfully!")
	return nil
}

