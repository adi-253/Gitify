package cmd

import (
	"fmt"
	"net/http"
	"os/exec"
    "runtime"
	"github.com/spf13/cobra"
	"time"
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
	Use: "spotify login",
	Short: "For logging into spotify",
	Run: func(cmd *cobra.Command, args []string) {

		go func() { // using go routine because if you were to write it after listenandserver it wont go to that line unless server is closed and when server is closed no use
			time.Sleep(1 * time.Second)
			openbrowser("http://localhost:8080/login")
    	}()


		http.HandleFunc("/login", LoginHandler)
		http.HandleFunc("/callback", HandleCallback)

		fmt.Println("Starting local server at http://localhost:8080/login ...")
		err := http.ListenAndServe(":8080", nil)
		if err != nil {
			fmt.Println("Server failed:", err)
    }
},
}

func init(){
	rootCmd.AddCommand(loginCmd)
}