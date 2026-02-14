package cmd

import (
	"fmt"

	"github.com/rnwolfe/mine/internal/version"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print mine version",
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Printf("mine %s\n", version.Full())
	},
}
