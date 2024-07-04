/*
Copyright © 2024 Dmitrii Petrov <dpetroff@gmail.com>
*/
package cmd

import (
	"github.com/can3p/blg/pkg/ops"
	"github.com/spf13/cobra"
)

func mergeCommand() *cobra.Command {
	out := &cobra.Command{
		Use:   "merge [id]",
		Short: "Merge missing fetched posts",
		Long:  `Check all fetched remote posts and in case they do not exist locally or were changed after creation create local files representing them.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := cmd.Flags().GetString("root")

			if err != nil {
				return err
			}

			return ops.OperationMerge(root)
		},
	}

	return out
}

func init() {
	rootCmd.AddCommand(mergeCommand())
}
