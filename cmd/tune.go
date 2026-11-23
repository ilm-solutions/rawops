package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func readKernelParam(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return "N/A (unavailable or restricted)"
	}
	return strings.TrimSpace(string(data))
}

var tuneCmd = &cobra.Command{
	Use:   "tune",
	Short: "Generate a reviewed Linux performance profile",
	RunE: func(cmd *cobra.Command, args []string) error {
		profile, _ := cmd.Flags().GetString("profile")
		apply, _ := cmd.Flags().GetBool("apply")
		if profile != "game" && profile != "vps" {
			return fmt.Errorf("--profile must be game or vps")
		}
		fmt.Printf("Current silicon : %s\n", GetSiliconVendor())
		fmt.Printf("Current swappiness: %s\n", readKernelParam("/proc/sys/vm/swappiness"))
		fmt.Printf("Current congestion: %s\n", readKernelParam("/proc/sys/net/ipv4/tcp_congestion_control"))

		var cfg string
		if profile == "game" {
			cfg = "# RAWOPS performance profile: game\nvm.swappiness = 1\nnet.core.default_qdisc = fq\nnet.ipv4.tcp_congestion_control = bbr\nnet.core.rmem_max = 16777216\nnet.core.wmem_max = 16777216\nnet.core.netdev_max_backlog = 16384\n"
		} else {
			cfg = "# RAWOPS performance profile: vps\nvm.swappiness = 10\nnet.core.default_qdisc = fq\nnet.ipv4.tcp_congestion_control = bbr\nfs.file-max = 2097152\nvm.dirty_ratio = 15\n"
		}
		if apply {
			return fmt.Errorf("live tuning is intentionally disabled in this foundation; review generated policy and use a future transactional apply release")
		}
		path := "rawops-tune-" + profile + ".conf"
		if err := os.WriteFile(path, []byte(cfg), 0600); err != nil {
			return err
		}
		fmt.Printf("Generated %s. No kernel settings were changed.\n", path)
		return nil
	},
}

func init() {
	tuneCmd.Flags().String("profile", "", "Performance profile (game | vps)")
	tuneCmd.Flags().Bool("apply", false, "Reserved for a future transactional executor; currently rejected")
	rootCmd.AddCommand(tuneCmd)
}
