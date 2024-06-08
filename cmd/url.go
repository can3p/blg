/*
Copyright © 2024 Dmitrii Petrov <dpetroff@gmail.com>
*/
package cmd

import (
	"github.com/spf13/cobra"
)

func urlCommand() *cobra.Command {
	out := &cobra.Command{
		Use:   "url <file>",
		Short: "Get url of a post",
		Long:  `Lookup url where file was published and print it`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return ErrNotImplemented
		},
	}

	return out
}

func init() {
	rootCmd.AddCommand(urlCommand())
}
