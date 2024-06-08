/*
Copyright © 2024 Dmitrii Petrov <dpetroff@gmail.com>
*/
package cmd

import (
	"github.com/spf13/cobra"
)

func markAsPulledCommand() *cobra.Command {
	out := &cobra.Command{
		Use:   "mark-as-pulled",
		Short: "Mark all posts as pulled from remote service",
		Long:  "Pretend that all local posts are uptodate comparing to the remote and save remote timestamps as a proof",
		RunE: func(cmd *cobra.Command, args []string) error {
			return ErrNotImplemented
		},
	}

	return out
}

func init() {
	rootCmd.AddCommand(markAsPulledCommand())
}
