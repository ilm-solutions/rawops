package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var shieldCmd = &cobra.Command{
	Use:   "shield",
	Short: "Generate or safely validate RAWOPS-owned nftables policy",
	RunE: func(cmd *cobra.Command, args []string) error {
		portsRaw, _ := cmd.Flags().GetString("ports")
		apply, _ := cmd.Flags().GetBool("apply")
		ports, err := parseShieldPorts(portsRaw)
		if err != nil {
			return err
		}
		joined := strings.Join(ports, ", ")
		table := fmt.Sprintf(`table inet rawops_shield {
    chain input {
        type filter hook input priority -10; policy accept;
        ct state invalid drop
        iifname "lo" accept
        ct state established,related accept
        tcp dport { %s } accept
        udp dport { %s } accept
    }
}
`, joined, joined)
		if !apply {
			path := "rawops-shield.nft"
			if err := os.WriteFile(path, []byte(table), 0600); err != nil {
				return err
			}
			fmt.Printf("Generated %s. No host changes were made.\n", path)
			return nil
		}
		if os.Geteuid() != 0 {
			return fmt.Errorf("--apply requires root")
		}
		if _, err := exec.LookPath("nft"); err != nil {
			return fmt.Errorf("nftables is not installed")
		}

		// Check syntax using a temporary table name so an existing RAWOPS table
		// cannot cause a false validation failure.
		checkRules := strings.Replace(table, "rawops_shield", "rawops_shield_check", 1)
		checkFile := "rawops-shield-check.nft"
		if err := os.WriteFile(checkFile, []byte(checkRules), 0600); err != nil {
			return err
		}
		defer os.Remove(checkFile)
		out, err := runCommand(15*time.Second, "nft", "-c", "-f", checkFile)
		if err != nil {
			return fmt.Errorf("nftables validation failed: %w: %s", err, strings.TrimSpace(string(out)))
		}

		// Snapshot before changing anything. Existing non-RAWOPS tables are untouched.
		if err := CreateHostSnapshot(); err != nil {
			return fmt.Errorf("snapshot failed; refusing apply: %w", err)
		}
		batch := table
		if _, listErr := runCommand(10*time.Second, "nft", "list", "table", "inet", "rawops_shield"); listErr == nil {
			batch = "delete table inet rawops_shield\n" + table
		}
		applyFile := "rawops-shield-apply.nft"
		if err := os.WriteFile(applyFile, []byte(batch), 0600); err != nil {
			return err
		}
		defer os.Remove(applyFile)
		out, err = runCommand(15*time.Second, "nft", "-c", "-f", applyFile)
		if err != nil {
			return fmt.Errorf("transaction preflight failed: %w: %s", err, strings.TrimSpace(string(out)))
		}
		out, err = runCommand(15*time.Second, "nft", "-f", applyFile)
		if err != nil {
			return fmt.Errorf("atomic nftables transaction failed: %w: %s", err, strings.TrimSpace(string(out)))
		}
		fmt.Println("RAWOPS-owned nftables table applied. Existing non-RAWOPS tables were not flushed.")
		return nil
	},
}

func init() {
	shieldCmd.Flags().String("ports", "22,25565", "Comma-separated TCP/UDP ports")
	shieldCmd.Flags().Bool("apply", false, "Apply RAWOPS-owned nftables table (requires root)")
	rootCmd.AddCommand(shieldCmd)
}
