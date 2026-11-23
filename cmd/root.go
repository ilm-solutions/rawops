package cmd

import (
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:           "rawops",
	Short:         "Production-conscious Linux infrastructure operations CLI",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		PrintBanner()
		fmt.Printf("[%sINF%s] Host: %s/%s | CPUs: %d | Silicon: %s\n",
			Cyan, Reset, runtime.GOOS, runtime.GOARCH, runtime.NumCPU(), GetSiliconVendor())
		return cmd.Help()
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Report verified local host capabilities",
	RunE: func(cmd *cobra.Command, args []string) error {
		PrintBanner()
		fmt.Printf("OS/ARCH       : %s/%s\n", runtime.GOOS, runtime.GOARCH)
		fmt.Printf("CPU cores     : %d\n", runtime.NumCPU())
		fmt.Printf("Silicon       : %s\n", GetSiliconVendor())
		fmt.Printf("Effective UID : %d\n", os.Geteuid())
		fmt.Printf("nft           : %s\n", commandPath("nft"))
		fmt.Printf("sysctl        : %s\n", commandPath("sysctl"))
		fmt.Printf("Config dir    : %s\n", "/etc/rawops")
		return nil
	},
}

func commandPath(name string) string {
	p, err := execLookPath(name)
	if err != nil {
		return "not installed"
	}
	return p
}

func Execute() {
	rootCmd.AddCommand(statusCmd, doctorCmd)
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "rawops:", err)
		os.Exit(1)
	}
}
