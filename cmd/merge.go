/*
Copyright © 2024 Dmitrii Petrov <dpetroff@gmail.com>
*/
package cmd

import (
	"github.com/spf13/cobra"
)

func mergeCommand() *cobra.Command {
	out := &cobra.Command{
		Use:   "merge [id]",
		Short: "Merge missing fetched posts",
		Long: `Check all fetched remote posts and in case they do not exist locally or were changed after creation create local files representing them.

If id is supplied only this post is going to be merged.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return ErrNotImplemented
		},
	}

	return out
}

func init() {
	rootCmd.AddCommand(mergeCommand())
}
