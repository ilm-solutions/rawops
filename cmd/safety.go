package cmd

import (
"fmt"
"os"
"os/exec"
"path/filepath"

"github.com/spf13/cobra"
)

const BackupDir = "/var/backups/rawops"

func CreateHostSnapshot() error {
if err := os.MkdirAll(BackupDir, 0700); err != nil {
return err
}
// 1. Backup current sysctl values
if out, err := exec.Command("sysctl", "-a").CombinedOutput(); err == nil {
_ = os.WriteFile(filepath.Join(BackupDir, "sysctl.bak"), out, 0600)
}
// 2. Backup active nftables ruleset
if out, err := exec.Command("nft", "list", "ruleset").CombinedOutput(); err == nil {
_ = os.WriteFile(filepath.Join(BackupDir, "nftables.bak"), out, 0600)
}
return nil
}

var rollbackCmd = &cobra.Command{
Use:   "rollback",
Short: "Revert kernel sysctl and nftables firewall to pre-RawOps snapshot",
Run: func(cmd *cobra.Command, args []string) {
PrintBanner()
if os.Geteuid() != 0 {
fmt.Printf("[%sERR%s] Root privileges (UID 0) required for system rollback.\n", Red+Bold, Reset)
return
}
RunTask("Removing /etc/sysctl.d/99-rawops.conf...", 600)
_ = os.Remove("/etc/sysctl.d/99-rawops.conf")
_ = exec.Command("sysctl", "--system").Run()

nftBak := filepath.Join(BackupDir, "nftables.bak")
if _, err := os.Stat(nftBak); err == nil {
RunTask("Restoring original nftables ruleset...", 700)
_ = exec.Command("nft", "flush", "ruleset").Run()
_ = exec.Command("nft", "-f", nftBak).Run()
}
fmt.Printf("\n[%sRESTORED%s] Host reverted to original baseline state.\n", Green+Bold, Reset)
},
}

func init() {
rootCmd.AddCommand(rollbackCmd)
}
