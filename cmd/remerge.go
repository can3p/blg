/*
Copyright © 2024 Dmitrii Petrov <dpetroff@gmail.com>
*/
package cmd

import (
	"github.com/spf13/cobra"
)

func remergeCommand() *cobra.Command {
	out := &cobra.Command{
		Use:   "remerge",
		Short: "Merge all the posts from the server even if they exist locally",
		Long: `Find posts that were already merged but not updated locally
afterwards and run merge again on them.

This functionality may be useful in case merge logic improves
and you want to get a better version of local posts.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return ErrNotImplemented
		},
	}

	return out
}

func init() {
	rootCmd.AddCommand(remergeCommand())
}
