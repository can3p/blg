/*
Copyright © 2024 Dmitrii Petrov <dpetroff@gmail.com>
*/
package cmd

import (
	"github.com/can3p/blg/pkg/ops"
	"github.com/spf13/cobra"
)

func newCommand() *cobra.Command {
	out := &cobra.Command{
		Use:   "new <name>",
		Short: "Start new post",
		Long:  `Open editor to edit a file like yyyy-mm-dd-<name>.md, where the date is a current date. if name has slashes like a/name file a/yyyy-mm-dd-name.md will be opened`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			open, err := cmd.Flags().GetBool("open")

			if err != nil {
				return err
			}

			root, err := cmd.Flags().GetString("root")

			if err != nil {
				return err
			}

			return ops.OperationNew(root, name, open)
		},
	}

	out.Flags().Bool("open", false, "open default editor to edit post")

	return out
}

func init() {
	rootCmd.AddCommand(newCommand())
}
