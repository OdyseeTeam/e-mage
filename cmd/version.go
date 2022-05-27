package cmd

import (
	"github.com/OdyseeTeam/e-mage/internal/version"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of E-Mage",
	Long:  `All software have versions. This is Mirage's`,
	Run: func(cmd *cobra.Command, args []string) {
		println(version.FullName())
	},
}
