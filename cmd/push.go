/*
Copyright © 2024 Dmitrii Petrov <dpetroff@gmail.com>
*/
package cmd

import (
	"github.com/can3p/blg/pkg/ops"
	"github.com/spf13/cobra"
)

func pushCommand() *cobra.Command {
	out := &cobra.Command{
		Use:   "push",
		Short: "Publish new posts",
		Long:  "Find new posts and publish them. Specify a filename if you want to publish a single file",
		Args:  cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := cmd.Flags().GetString("root")

			if err != nil {
				return err
			}

			var singleFile string

			if len(args) == 1 {
				singleFile = args[0]
			}

			return ops.OperationPush(root, singleFile)
		},
	}

	return out
}

func init() {
	rootCmd.AddCommand(pushCommand())
}
