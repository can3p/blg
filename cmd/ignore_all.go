/*
Copyright © 2024 Dmitrii Petrov <dpetroff@gmail.com>
*/
package cmd

import (
	"github.com/spf13/cobra"
)

func ignoreAllCommand() *cobra.Command {
	out := &cobra.Command{
		Use:   "ignore-all",
		Short: "Mark all posts as up to date locally",
		Long:  "Sometimes timestamps on files go out of sync with database and no real update is necessary. This command will update database to treat all posts as up to date",
		RunE: func(cmd *cobra.Command, args []string) error {
			return ErrNotImplemented
		},
	}

	return out
}

func init() {
	rootCmd.AddCommand(ignoreAllCommand())
}
