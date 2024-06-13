package cmd

import (
	"os"

	"github.com/can3p/blg/generated/buildinfo"
	cmd "github.com/can3p/kleiner/shared/cmd/cobra"
	"github.com/can3p/kleiner/shared/published"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var ErrNotImplemented = errors.Errorf("Command has not been implemented")

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "blg",
	Short: "Blogging client",
	Long:  `Command line blogging client`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	info := buildinfo.Info()
	cwd, err := os.Getwd()

	if err != nil {
		panic(err)
	}

	rootCmd.PersistentFlags().String("root", cwd, "Specify the folder to work against, defaults to current folder")

	cmd.Setup(info, rootCmd)
	published.MaybeNotifyAboutNewVersion(info)
}
