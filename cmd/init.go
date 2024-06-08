/*
Copyright © 2024 Dmitrii Petrov <dpetroff@gmail.com>
*/
package cmd

import (
	"github.com/spf13/cobra"
)

func initCommand() *cobra.Command {
	out := &cobra.Command{
		Use:   "init <service_name>",
		Short: "Init folder to start posting",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return ErrNotImplemented
		},
	}

	return out
}

func init() {
	rootCmd.AddCommand(initCommand())
}
