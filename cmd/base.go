package cmd

import (
	"github.com/spf13/cobra"
)

var spotifyCmd = &cobra.Command{
	Use: "spotify",
	Short: "Base command for all spotify commands",
}

func init(){
	rootCmd.AddCommand(spotifyCmd)
}