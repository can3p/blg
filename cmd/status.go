/*
Copyright © 2024 Dmitrii Petrov <dpetroff@gmail.com>
*/
package cmd

import (
	"github.com/spf13/cobra"
)

func statusCommand() *cobra.Command {
	out := &cobra.Command{
		Use:   "status",
		Short: "Display state of out of sync files",
		Long:  "Display the list with filenames of all drafts or files that are out of sync with blog service.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return ErrNotImplemented
		},
	}

	return out
}

func init() {
	rootCmd.AddCommand(statusCommand())
}
