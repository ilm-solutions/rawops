package cmd

import (
"fmt"
"os"
"os/exec"
"strings"

"github.com/spf13/cobra"
)

type ToolModule struct {
ID          string
Category    string
PackageName string
Role        string
}

var ArsenalRegistry = []ToolModule{
{"net-ethtool", "NET", "ethtool", "NIC Driver & Ring-Buffer Control"},
{"net-iperf3", "NET", "iperf3", "Raw Bandwidth Benchmarking"},
{"net-mtr", "NET", "mtr-tiny", "Real-Time Packet Loss Tracer"},
{"net-tcpdump", "NET", "tcpdump", "Raw Packet Sniffer"},
{"net-wg", "NET", "wireguard-tools", "Kernel-Space Mesh VPN"},
{"net-tc", "NET", "iproute2", "Traffic Control & QoS Shaping"},
{"sec-nft", "SEC", "nftables", "In-Kernel Packet Filter"},
{"sec-f2b", "SEC", "fail2ban", "Automated Intrusion Blocker"},
{"sec-audit", "SEC", "auditd", "Kernel Syscall Auditing"},
{"sec-lynis", "SEC", "lynis", "Deep Security Hardening Audit"},
{"virt-docker", "VIRT", "docker.io", "Container Runtime Engine"},
{"virt-kvm", "VIRT", "qemu-kvm", "Type-1 Bare-Metal Hypervisor"},
{"hw-cpupower", "HW", "linux-cpupower", "Silicon Frequency Governor"},
{"hw-smart", "HW", "smartmontools", "NVMe/SSD Health Telemetry"},
{"hw-numa", "HW", "numactl", "NUMA Memory & Core Pinning"},
{"hw-nvme", "HW", "nvme-cli", "Direct NVMe Controller Management"},
{"mon-btop", "MON", "btop", "Live Hardware Resource Monitor"},
{"mon-sysstat", "MON", "sysstat", "Kernel Performance Counters"},
}

var BundlePresets = map[string][]string{
"game-host": {"net-ethtool", "net-iperf3", "sec-nft", "sec-f2b", "hw-cpupower", "hw-numa", "mon-btop", "mon-sysstat"},
"vps-node":  {"virt-kvm", "virt-docker", "net-wg", "sec-nft", "sec-audit", "hw-smart", "hw-nvme", "mon-sysstat"},
"sec-core":  {"sec-nft", "sec-f2b", "sec-audit", "sec-lynis", "net-tcpdump"},
}

func detectPkgManager() string {
managers := []string{"apt-get", "dnf", "yum", "pacman", "apk"}
for _, m := range managers {
if _, err := exec.LookPath(m); err == nil {
return m
}
}
return "unknown"
}

func resolvePackage(id string) (ToolModule, bool) {
for _, t := range ArsenalRegistry {
if t.ID == id {
return t, true
}
}
return ToolModule{}, false
}

var arsenalCmd = &cobra.Command{
Use:   "arsenal",
Short: "Manage, search, and deploy bare-metal hosting tools & bundles",
}

var arsenalListCmd = &cobra.Command{
Use:   "list",
Short: "Display all indexed bare-metal modules",
Run: func(cmd *cobra.Command, args []string) {
PrintBanner()
fmt.Printf("[%sINF%s] Indexed Modules: %d | Bundles: %d | PkgMgr: %s\n\n", Cyan, Reset, len(ArsenalRegistry), len(BundlePresets), detectPkgManager())
fmt.Printf("%-14s %-6s %-18s %s\n", "MODULE_ID", "CAT", "PACKAGE", "SYSTEM_ROLE")
fmt.Println("------------------------------------------------------------------------")
for _, t := range ArsenalRegistry {
fmt.Printf("%s%-14s%s %-6s %-18s %s\n", Bold, t.ID, Reset, t.Category, t.PackageName, t.Role)
}
},
}

var arsenalDeployCmd = &cobra.Command{
Use:   "deploy [module-id]",
Short: "Provision and install a single module onto the host system",
Args:  cobra.ExactArgs(1),
Run: func(cmd *cobra.Command, args []string) {
execLive, _ := cmd.Flags().GetBool("exec")
target := strings.ToLower(args[0])
PrintBanner()
pkgMgr := detectPkgManager()

t, ok := resolvePackage(target)
if !ok {
fmt.Printf("[%sERR%s] Module '%s' not found in Arsenal Registry.\n", Red+Bold, Reset, target)
return
}

fmt.Printf("[%sTARGET%s] %s (%s) via [%s]\n", Cyan, Reset, t.ID, t.PackageName, pkgMgr)
RunTask("Preparing package pipeline for "+t.PackageName+"...", 700)

if !execLive {
fmt.Printf("[%sDRY-RUN%s] Ready: %s install -y %s (Add --exec for live install)\n", Yellow, Reset, pkgMgr, t.PackageName)
return
}
if os.Geteuid() != 0 {
fmt.Printf("[%sERR%s] Root privileges (UID 0) required for live package installation.\n", Red+Bold, Reset)
return
}
c := exec.Command(pkgMgr, "install", "-y", t.PackageName)
c.Stdout = os.Stdout
c.Stderr = os.Stderr
_ = c.Run()
},
}

var arsenalBundleCmd = &cobra.Command{
Use:   "bundle [game-host|vps-node|sec-core]",
Short: "Deploy a complete hosting preset stack in one command",
Args:  cobra.ExactArgs(1),
Run: func(cmd *cobra.Command, args []string) {
preset := strings.ToLower(args[0])
PrintBanner()
modules, exists := BundlePresets[preset]
if !exists {
fmt.Printf("[%sERR%s] Unknown bundle '%s'. Available: game-host, vps-node, sec-core\n", Red+Bold, Reset, preset)
return
}
fmt.Printf("[%sBUNDLE%s] Initializing [%s] stack (%d modules)...\n\n", Cyan+Bold, Reset, strings.ToUpper(preset), len(modules))
for i, modID := range modules {
if t, ok := resolvePackage(modID); ok {
fmt.Printf("[%02d/%02d] Queueing %-14s -> PKG: %s\n", i+1, len(modules), t.ID, t.PackageName)
}
}
fmt.Printf("\n[%sREADY%s] Bundle pipeline verified. Run as root with --exec on x86_64 host.\n", Green+Bold, Reset)
},
}

func init() {
arsenalDeployCmd.Flags().Bool("exec", false, "Execute live system package installation")
arsenalCmd.AddCommand(arsenalListCmd, arsenalDeployCmd, arsenalBundleCmd)
rootCmd.AddCommand(arsenalCmd)
}
