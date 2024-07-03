/*
Copyright © 2024 Dmitrii Petrov <dpetroff@gmail.com>
*/
package cmd

import (
	"github.com/can3p/blg/pkg/ops"
	"github.com/spf13/cobra"
)

func lastCommand() *cobra.Command {
	out := &cobra.Command{
		Use:   "last",
		Short: "Edit the last post",
		Long:  "Open editor to edit last published post",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := cmd.Flags().GetString("root")

			if err != nil {
				return err
			}

			return ops.OperationLast(root)
		},
	}

	return out
}

func init() {
	rootCmd.AddCommand(lastCommand())
}
