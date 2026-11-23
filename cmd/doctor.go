package cmd

import (
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check prerequisites without changing host state",
	RunE: func(cmd *cobra.Command, args []string) error {
		PrintBanner()
		checks := []struct {
			name   string
			ok     bool
			detail string
		}{
			{"Linux host", runtime.GOOS == "linux", runtime.GOOS},
			{"64-bit architecture", runtime.GOARCH == "amd64" || runtime.GOARCH == "arm64", runtime.GOARCH},
			{"nftables binary", commandExists("nft"), commandPath("nft")},
			{"sysctl binary", commandExists("sysctl"), commandPath("sysctl")},
			{"root privileges", os.Geteuid() == 0, fmt.Sprintf("UID %d", os.Geteuid())},
		}
		failed := 0
		for _, c := range checks {
			state := "OK"
			if !c.ok {
				state = "WARN"
				failed++
			}
			fmt.Printf("[%-4s] %-24s %s\n", state, c.name, c.detail)
		}
		if failed > 0 {
			return fmt.Errorf("%d prerequisite check(s) need attention", failed)
		}
		fmt.Println("All prerequisite checks passed. No host changes were made.")
		return nil
	},
}

func commandExists(name string) bool { _, err := execLookPath(name); return err == nil }
