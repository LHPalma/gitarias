package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:           "gtr",
	Short:         "Utilitários de git pro dia a dia",
	Long:          "gitarias (gtr) — utilitários para as tarefas repetitivas de git local.",
	SilenceErrors: true,
	SilenceUsage:  true,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(rootCmd.ErrOrStderr(), "erro:", err)
		os.Exit(1)
	}
}
