package cmd

import (
"fmt"
"os/exec"
"strings"

"github.com/spf13/cobra"
)

type ToolModule struct {
ID          string
Category    string
PackageName string
Role        string
Description string
}

// Scalable Arsenal Registry (Add 500-1000+ tools here)
var ArsenalRegistry = []ToolModule{
{"net-ethtool", "NET", "ethtool", "NIC Driver & Ring-Buffer Control", "Inspect and tune network card hardware queues"},
{"net-iperf3", "NET", "iperf3", "Raw Bandwidth Benchmarking", "Measure max TCP/UDP throughput between nodes"},
{"net-mtr", "NET", "mtr-tiny", "Real-Time Packet Loss Tracer", "Diagnose BGP routing and latency spikes"},
{"net-tcpdump", "NET", "tcpdump", "Raw Packet Sniffer", "Capture and inspect live L2-L4 network frames"},
{"net-wg", "NET", "wireguard-tools", "Kernel-Space Mesh VPN", "Zero-overhead encrypted node-to-node tunneling"},
{"sec-nft", "SEC", "nftables", "In-Kernel Packet Filter", "High-speed stateful firewall and NAT engine"},
{"sec-f2b", "SEC", "fail2ban", "Automated Intrusion Blocker", "Ban brute-force IPs at firewall level"},
{"sec-audit", "SEC", "auditd", "Kernel Syscall Auditing", "Track unauthorized root and file modifications"},
{"sec-lynis", "SEC", "lynis", "Deep Security Hardening Audit", "Scan Linux kernel and system vulnerabilities"},
{"virt-docker", "VIRT", "docker.io", "Container Runtime Engine", "Isolated microservice and game-server containers"},
{"virt-kvm", "VIRT", "qemu-kvm", "Type-1 Bare-Metal Hypervisor", "Hardware-assisted x86-64 VPS virtualization"},
{"hw-cpupower", "HW", "linux-cpupower", "Silicon Frequency Governor", "Lock CPU cores to max boost clock speed"},
{"hw-smart", "HW", "smartmontools", "NVMe/SSD Health Telemetry", "Read raw disk wear-leveling and I/O errors"},
{"hw-numa", "HW", "numactl", "NUMA Memory & Core Pinning", "Bind processes to local CPU socket memory"},
{"mon-btop", "MON", "btop", "Live Hardware Resource Monitor", "Real-time CPU, RAM, disk, and net telemetry"},
{"mon-sysstat", "MON", "sysstat", "Kernel Performance Counters", "iostat, mpstat, and sar low-level metrics"},
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

var arsenalCmd = &cobra.Command{
Use:   "arsenal",
Short: "Manage, search, and deploy bare-metal hosting tools",
}

var arsenalListCmd = &cobra.Command{
Use:   "list",
Short: "Display all indexed bare-metal modules",
Run: func(cmd *cobra.Command, args []string) {
PrintBanner()
fmt.Printf("[%sINF%s] Indexed Arsenal Modules: %d | Host PkgMgr: %s\n\n", Cyan, Reset, len(ArsenalRegistry), detectPkgManager())
fmt.Printf("%-14s %-6s %-18s %s\n", "MODULE_ID", "CAT", "PACKAGE", "SYSTEM_ROLE")
fmt.Println("------------------------------------------------------------------------")
for _, t := range ArsenalRegistry {
fmt.Printf("%s%-14s%s %-6s %-18s %s\n", Bold, t.ID, Reset, t.Category, t.PackageName, t.Role)
}
},
}

var arsenalSearchCmd = &cobra.Command{
Use:   "search [keyword]",
Short: "Search tools by ID, category, or capability",
Args:  cobra.ExactArgs(1),
Run: func(cmd *cobra.Command, args []string) {
query := strings.ToLower(args[0])
PrintBanner()
found := 0
for _, t := range ArsenalRegistry {
if strings.Contains(strings.ToLower(t.ID+t.Category+t.Role+t.Description), query) {
fmt.Printf("[%sMATCH%s] %-14s | PKG: %-16s | %s\n", Green, Reset, t.ID, t.PackageName, t.Role)
found++
}
}
fmt.Printf("\n[%sINF%s] Total matches found: %d\n", Cyan, Reset, found)
},
}

var arsenalDeployCmd = &cobra.Command{
Use:   "deploy [module-id]",
Short: "Provision and install a module onto the host system",
Args:  cobra.ExactArgs(1),
Run: func(cmd *cobra.Command, args []string) {
target := strings.ToLower(args[0])
PrintBanner()
pkgMgr := detectPkgManager()
for _, t := range ArsenalRegistry {
if t.ID == target {
fmt.Printf("[%sTARGET%s] %s (%s) via [%s]\n", Cyan, Reset, t.ID, t.PackageName, pkgMgr)
RunTask("Resolving dependencies & preparing "+t.PackageName+"...", 900)
if pkgMgr == "unknown" {
fmt.Printf("[%sWARN%s] No standard Linux root package manager detected (Dry-Run Mode).\n", Yellow, Reset)
return
}
fmt.Printf("[%sEXEC%s] Command Ready: %s install -y %s\n", Green+Bold, Reset, pkgMgr, t.PackageName)
return
}
}
fmt.Printf("[%sERR%s] Module '%s' not found in Arsenal Registry.\n", Red+Bold, Reset, target)
},
}

func init() {
arsenalCmd.AddCommand(arsenalListCmd, arsenalSearchCmd, arsenalDeployCmd)
rootCmd.AddCommand(arsenalCmd)
}
