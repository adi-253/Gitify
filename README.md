# Gitify

A terminal-first Spotify CLI/TUI powered by Bubble Tea. Gitify manages login, playlists, and playback through the Spotify Web API.

## Quick setup

1. **Register at the Spotify Developer Dashboard**
   - Visit https://developer.spotify.com/documentation/web-api and sign in with your Spotify account.
   - Create an app, note the **Client ID** and **Client Secret**, and set the **Redirect URI** to `http://127.0.0.1:8080/callback` (must match `REDIRECT_URL`).
2. **Create a `.env` file in the project root** (ignored by git) and add the values you just collected:
   ```env
   CLIENT_ID=<your-client-id>
   CLIENT_SECRET=<your-client-secret>
   REDIRECT_URL=http://127.0.0.1:8080/callback
   ```
3. **Install Go 1.25.3** (or later) and download module dependencies:
   ```bash
   go mod download
   ```

## Running Gitify

- Authenticate with Spotify:

  ```bash
  go run main.go spotify login
  ```

  This starts a local server, opens your browser, and stores the tokens in `token.json`/`profile.json`.

- Launch the TUI:

  ```bash
  go run main.go spotify tui
  ```

- Use the CLI when you do not want the TUI:

  ```bash
  go run main.go spotify playlist
  go run main.go spotify search "Song Title"
  go run main.go spotify me
  go run main.go spotify pause|resume|next|prev
  ```

- Spotify Premium is required for playback control and streaming endpoints.


## Notes

- Tokens/credentials are stored in `token.json` and `profile.json` (`.gitignore`-d).
- The app automatically refreshes the access token when expired.
- Also for playing it on the device you want , spotify should be open in that device and also play and pause once 
  so that the device gets recognized by the Gitify to play the song there.
