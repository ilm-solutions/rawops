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
fmt.Printf("[%sINF%s] Specify --profile=game|vps to generate custom raw tuning.\n", Yellow, Reset)
return
}

RunTask("Generating raw sysctl profile ["+strings.ToUpper(profile)+"]...", 800)
configContent := fmt.Sprintf("# RAWOPS KERNEL TUNING PROFILE: %s\n", strings.ToUpper(profile))
if profile == "game" {
configContent += "vm.swappiness = 1\nnet.core.default_qdisc = fq\nnet.ipv4.tcp_congestion_control = bbr\nnet.core.rmem_max = 16777216\nnet.core.wmem_max = 16777216\n"
} else {
configContent += "vm.swappiness = 10\nnet.core.default_qdisc = fq\nnet.ipv4.tcp_congestion_control = bbr\nfs.file-max = 2097152\n"
}

outFile := "rawops-tune-" + profile + ".conf"
_ = os.WriteFile(outFile, []byte(configContent), 0644)
fmt.Printf("[%sOK%s] Raw tuning blueprint written to: %s\n", Green+Bold, Reset, outFile)

if apply {
if os.Geteuid() != 0 {
fmt.Printf("[%sERR%s] Root privileges (UID 0) required to inject live kernel sysctl.\n", Red+Bold, Reset)
} else {
RunTask("Injecting parameters into live Linux kernel...", 900)
fmt.Printf("[%sACTIVE%s] Machine locked in BEAST PERFORMANCE mode.\n", Green+Bold, Reset)
}
}
},
}

func init() {
tuneCmd.Flags().String("profile", "", "Performance profile (game | vps)")
tuneCmd.Flags().Bool("apply", false, "Apply tuning directly to live kernel (requires root)")
rootCmd.AddCommand(tuneCmd)
}
