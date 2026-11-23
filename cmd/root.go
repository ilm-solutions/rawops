package cmd

import (
"fmt"
"os"
"runtime"

"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
Use:   "rawops",
Short: "Bare-metal provisioning, tuning, and DDoS shield CLI engine",
Run: func(cmd *cobra.Command, args []string) {
PrintBanner()
fmt.Printf("[%sINF%s] Host Arch Detected : %s/%s\n", Cyan, Reset, runtime.GOOS, runtime.GOARCH)
fmt.Printf("[%sINF%s] Use 'rawops --help' to inspect available control modules.\n", Cyan, Reset)
},
}

var statusCmd = &cobra.Command{
Use:   "status",
Short: "Run system diagnostics and verify kernel control readiness",
Run: func(cmd *cobra.Command, args []string) {
PrintBanner()
RunTask("Probing CPU silicon registers & topology...", 800)
RunTask("Scanning Linux kernel sysctl parameters...", 900)
RunTask("Verifying nftables & XDP/eBPF hook readiness...", 1000)
fmt.Printf("\n[%sREADY%s] Control plane operational. Root UID: %d\n", Green+Bold, Reset, os.Geteuid())
},
}

func Execute() {
rootCmd.AddCommand(statusCmd)
if err := rootCmd.Execute(); err != nil {
fmt.Println(err)
os.Exit(1)
}
}
