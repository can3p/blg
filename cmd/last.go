/*
Copyright © 2024 Dmitrii Petrov <dpetroff@gmail.com>
*/
package cmd

import (
	"github.com/spf13/cobra"
)

func lastCommand() *cobra.Command {
	out := &cobra.Command{
		Use:   "last",
		Short: "Edit the last post",
		Long:  "Open editor to edit last published post",
		RunE: func(cmd *cobra.Command, args []string) error {
			return ErrNotImplemented
		},
	}

	return out
}

func init() {
	rootCmd.AddCommand(lastCommand())
}
