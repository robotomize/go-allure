package main

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"strings"

	"github.com/spf13/cobra"
)

var (
	BuildName = "golurectl"
	BuildTag  string
)

var versionCmd = &cobra.Command{
	Use:          "version",
	Long:         "Print actual version",
	Short:        "actual version",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if _, err := cmd.OutOrStdout().Write([]byte(version())); err != nil {
			return err
		}

		return nil
	},
}

func init() {
	if info, available := debug.ReadBuildInfo(); available {
		if BuildTag == "" {
			BuildTag = info.Main.Version
		}
	}

	rootCmd.AddCommand(versionCmd)
}

func version() string {
	return fmt.Sprintf(
		"%s version %s %s %s", BuildName, strings.Replace(BuildTag, "v", "", -1), runtime.GOOS, runtime.GOARCH,
	)
}
