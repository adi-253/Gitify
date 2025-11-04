package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"runtime"
	"time"

	"github.com/adi-253/Gitify/cmd/utils"
	"github.com/spf13/cobra"
)

func openbrowser(url string) {
    var err error
    switch runtime.GOOS {
    case "linux":
        err = exec.Command("xdg-open", url).Start()
    case "windows":
        err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
    case "darwin":
        err = exec.Command("open", url).Start()
    }
    if err != nil {
        fmt.Println("Please open the following URL manually:", url)
    }
}

var loginCmd = &cobra.Command{
	Use: "login",
	Short: "For logging into spotify",
	Run: func(cmd *cobra.Command, args []string) {
		// Create a channel to signal when login is complete
		done := make(chan bool)

		// Create a new ServeMux to avoid conflicts with global handlers
		mux := http.NewServeMux()
		
		// Create server with context for graceful shutdown
		server := &http.Server{
			Addr:    ":8080",
			Handler: mux,
		}

		// Setup handlers with the done channel
		mux.HandleFunc("/login", utils.LoginHandler)
		mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
			utils.HandleCallback(w, r)
			// Signal that login is complete
			go func() {
				time.Sleep(1 * time.Second) // Give time for response to be sent
				done <- true
			}()
		})

		// Start server in goroutine
		go func() {
			fmt.Println("Starting local server at http://localhost:8080/login ...")
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				fmt.Println("Server failed:", err)
			}
		}()

		// Open browser after server starts
		go func() {
			time.Sleep(1 * time.Second)
			openbrowser("http://localhost:8080/login")
		}()

		// Wait for login completion
		<-done

		// Shutdown server gracefully
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		fmt.Println("Login completed. Shutting down server...")
		if err := server.Shutdown(ctx); err != nil {
			fmt.Printf("Server shutdown error: %v\n", err)
		}

		//Also run storing profile info
		err:=getUserInfo()
		if err!=nil{
			fmt.Print("Couldnt fetch profile info")
		}
		fmt.Printf("Successfuly stored user profile")
	},
}

func init(){
	spotfiyCmd.AddCommand(loginCmd)
}