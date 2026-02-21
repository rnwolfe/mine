package cmd

import (
	"fmt"

	"github.com/rnwolfe/mine/internal/hook"
	"github.com/rnwolfe/mine/internal/version"
	"github.com/spf13/cobra"
)

var (
	versionShort bool
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print mine version",
	RunE:  hook.Wrap("version", runVersion),
}

func runVersion(_ *cobra.Command, _ []string) error {
	if versionShort {
		fmt.Println(version.Short())
	} else {
		fmt.Printf("mine %s\n", version.Full())
	}
	return nil
}

func init() {
	versionCmd.Flags().BoolVar(&versionShort, "short", false, "Print only the version number")
}
