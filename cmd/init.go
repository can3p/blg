/*
Copyright © 2024 Dmitrii Petrov <dpetroff@gmail.com>
*/
package cmd

import (
	"fmt"
	"strings"

	"github.com/can3p/blg/pkg/ops"
	"github.com/can3p/blg/pkg/types"
	"github.com/spf13/cobra"
)

func initCommand() *cobra.Command {
	out := &cobra.Command{
		Use:   "init <service_name>",
		Short: fmt.Sprintf("Init folder to start posting, possible services: %s", strings.Join(types.DefaultServiceRepo.Services(), ", ")),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			serviceName := args[0]

			host, err := cmd.Flags().GetString("host")

			if err != nil {
				return err
			}

			root, err := cmd.Flags().GetString("root")

			if err != nil {
				return err
			}

			return ops.OperationInit(serviceName, host, root)
		},
	}

	out.Flags().String("host", "", "Specify in case the service instance runs on a custom host")

	return out
}

func init() {
	rootCmd.AddCommand(initCommand())
}
