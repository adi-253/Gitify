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

- Inspect playlists, playback, search, and profile via the CLI commands (e.g., `spotify playlist`, `spotify search <song>`).

## Notes

- Tokens/credentials are stored in `token.json` and `profile.json` (`.gitignore`-d).
- The app automatically refreshes the access token when expired.
- Silent playback mode is enabled when running the Bubble Tea TUI to keep the UI clean.
