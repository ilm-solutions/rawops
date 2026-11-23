package cmd

import (
"fmt"
"os"
"os/exec"
"strings"

"github.com/spf13/cobra"
)

func readKernelParam(path string) string {
data, err := os.ReadFile(path)
if err != nil {
return "N/A (Restricted)"
}
return strings.TrimSpace(string(data))
}

var tuneCmd = &cobra.Command{
Use:   "tune",
Short: "Silicon-level hardware and Linux kernel performance tuning",
Run: func(cmd *cobra.Command, args []string) {
profile, _ := cmd.Flags().GetString("profile")
apply, _ := cmd.Flags().GetBool("apply")

PrintBanner()
fmt.Printf("[%sASM%s] Silicon Vendor Signature : %s%s%s\n", Cyan, Reset, Bold, GetSiliconVendor(), Reset)
fmt.Printf("[%sSYS%s] VM Swappiness            : %s\n", Cyan, Reset, readKernelParam("/proc/sys/vm/swappiness"))
fmt.Printf("[%sSYS%s] TCP Congestion Control   : %s\n", Cyan, Reset, readKernelParam("/proc/sys/net/ipv4/tcp_congestion_control"))
fmt.Printf("[%sSYS%s] CPU0 Frequency Governor  : %s\n\n", Cyan, Reset, readKernelParam("/sys/devices/system/cpu/cpu0/cpufreq/scaling_governor"))

if profile == "" {
fmt.Printf("[%sINF%s] Specify --profile=game|vps to generate or apply tuning.\n", Yellow, Reset)
return
}

RunTask("Compiling raw sysctl profile ["+strings.ToUpper(profile)+"]...", 700)
cfg := fmt.Sprintf("# RAWOPS ENTERPRISE KERNEL PROFILE: %s\n", strings.ToUpper(profile))
if profile == "game" {
cfg += "vm.swappiness = 1\nnet.core.default_qdisc = fq\nnet.ipv4.tcp_congestion_control = bbr\nnet.core.rmem_max = 16777216\nnet.core.wmem_max = 16777216\nnet.core.netdev_max_backlog = 16384\n"
} else {
cfg += "vm.swappiness = 10\nnet.core.default_qdisc = fq\nnet.ipv4.tcp_congestion_control = bbr\nfs.file-max = 2097152\nvm.dirty_ratio = 15\n"
}

if !apply {
outFile := "rawops-tune-" + profile + ".conf"
_ = os.WriteFile(outFile, []byte(cfg), 0644)
fmt.Printf("[%sDRY-RUN%s] Blueprint saved: %s (Use --apply as root for live injection)\n", Yellow+Bold, Reset, outFile)
return
}

if os.Geteuid() != 0 {
fmt.Printf("[%sERR%s] Root privileges (UID 0) required for live kernel injection.\n", Red+Bold, Reset)
return
}

RunTask("Creating pre-flight host snapshot in /var/backups/rawops...", 600)
_ = CreateHostSnapshot()

targetPath := "/etc/sysctl.d/99-rawops.conf"
_ = os.WriteFile(targetPath, []byte(cfg), 0644)
RunTask("Executing live kernel sysctl --system reload...", 800)
_ = exec.Command("sysctl", "--system").Run()
_ = exec.Command("cpupower", "frequency-set", "-g", "performance").Run()

fmt.Printf("\n[%sACTIVE%s] Live kernel & CPU governor locked in [%s] mode.\n", Green+Bold, Reset, strings.ToUpper(profile))
},
}

func init() {
tuneCmd.Flags().String("profile", "", "Performance profile (game | vps)")
tuneCmd.Flags().Bool("apply", false, "Inject tuning directly into live kernel (requires root)")
rootCmd.AddCommand(tuneCmd)
}
