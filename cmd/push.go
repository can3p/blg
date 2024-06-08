/*
Copyright © 2024 Dmitrii Petrov <dpetroff@gmail.com>
*/
package cmd

import (
	"github.com/spf13/cobra"
)

func pushCommand() *cobra.Command {
	out := &cobra.Command{
		Use:   "push",
		Short: "Publish new posts",
		Long:  "Find new posts and publish them. Drafts will be skipped client will prompt before publishing every file.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return ErrNotImplemented
		},
	}

	return out
}

func init() {
	rootCmd.AddCommand(pushCommand())
}
