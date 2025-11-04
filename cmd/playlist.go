package cmd

import(
	"github.com/spf13/cobra"
)

type playlistInfo struct{

}

var playlistCmd = &cobra.Command{
	Use: "show playlist",
}