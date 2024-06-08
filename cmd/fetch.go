/*
Copyright © 2024 Dmitrii Petrov <dpetroff@gmail.com>
*/
package cmd

import (
	"github.com/spf13/cobra"
)

func fetchCommand() *cobra.Command {
	out := &cobra.Command{
		Use:   "fetch",
		Short: "Fetch posts from the server",
		Long:  "Fetch all the posts from the server from the moment of last update. This action will not merge posts.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return ErrNotImplemented
		},
	}

	return out
}

func init() {
	rootCmd.AddCommand(fetchCommand())
}
