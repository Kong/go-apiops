package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// VERSION is the current version of decK.
// This should be substituted by git tag during the build process.
var VERSION = "dev"

// COMMIT is the short hash of the source tree.
// This should be substituted by Git commit hash during the build process.
var COMMIT = "unknown"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the kceD version",
	Long: `The version command prints the version of kceD along with a Git short
commit hash of the source tree.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("kceD %s (%s) \n", VERSION, COMMIT)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
