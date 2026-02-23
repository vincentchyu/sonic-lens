package main

import (
	"github.com/spf13/cobra"
)

func NewCommand(use, short, long string) *cobra.Command {
	return &cobra.Command{
		Use:                   use,
		Long:                  long,
		Short:                 short,
		SilenceErrors:         true,
		SilenceUsage:          true,
		TraverseChildren:      true,
		DisableFlagsInUseLine: true,
	}
}
