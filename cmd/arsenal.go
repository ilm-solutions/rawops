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
{"net-iperf3", "NET", "iperf3", "Raw TCP/UDP Bandwidth Benchmarking"},
{"net-mtr", "NET", "mtr-tiny", "Real-Time BGP & Packet Loss Tracer"},
{"net-tcpdump", "NET", "tcpdump", "Raw L2-L4 Packet Capture Analyzer"},
{"net-wg", "NET", "wireguard-tools", "Kernel-Space Zero-Overhead Mesh VPN"},
{"net-tc", "NET", "iproute2", "Traffic Control & QoS Packet Shaping"},
{"net-haproxy", "NET", "haproxy", "High-Concurrency L4/L7 Load Balancer"},
{"net-nginx", "NET", "nginx", "Reverse Proxy & Edge Web Server"},
{"net-bird2", "NET", "bird2", "BGP / OSPF Anycast Routing Daemon"},
{"net-frr", "NET", "frr", "Free Range Routing Data Center Suite"},
{"net-unbound", "NET", "unbound", "Recursive Caching DNS Resolver"},
{"net-nmap", "NET", "nmap", "Network Port & Topology Scanner"},
{"sec-nft", "SEC", "nftables", "In-Kernel Netfilter Packet Filter"},
{"sec-ipset", "SEC", "ipset", "O(1) Hash-Based IP Blacklist Engine"},
{"sec-f2b", "SEC", "fail2ban", "Automated Brute-Force Blocker"},
{"sec-audit", "SEC", "auditd", "Kernel Syscall & Security Auditing"},
{"sec-lynis", "SEC", "lynis", "Deep OS Hardening & Audit Scanner"},
{"sec-aide", "SEC", "aide", "Filesystem Integrity & Rootkit Tripwire"},
{"sec-rkhunter", "SEC", "rkhunter", "Rootkit & Local Exploit Scanner"},
{"sec-clamav", "SEC", "clamav", "Malware & Payload Scanner"},
{"virt-kvm", "VIRT", "qemu-kvm", "Type-1 Bare-Metal Hypervisor"},
{"virt-libvirt", "VIRT", "libvirt-daemon-system", "Enterprise VM Lifecycle API"},
{"virt-docker", "VIRT", "docker.io", "OCI Container Runtime Engine"},
{"virt-podman", "VIRT", "podman", "Daemonless Rootless Container Engine"},
{"virt-lxc", "VIRT", "lxc", "Lightweight OS-Level Containers"},
{"virt-cloudinit", "VIRT", "cloud-init", "Automated VM Instance Provisioning"},
{"hw-cpupower", "HW", "linux-cpupower", "CPU Frequency Governor Lock"},
{"hw-numa", "HW", "numactl", "NUMA Socket Memory & Core Pinning"},
{"hw-smart", "HW", "smartmontools", "NVMe/SSD SMART Health Telemetry"},
{"hw-nvme", "HW", "nvme-cli", "Direct PCIe NVMe Controller CLI"},
{"hw-sensors", "HW", "lm-sensors", "Motherboard VRM & CPU Thermal Telemetry"},
{"hw-dmidecode", "HW", "dmidecode", "SMBIOS Hardware & RAM Decoder"},
{"stor-zfs", "STOR", "zfsutils-linux", "Enterprise Copy-on-Write ZFS Engine"},
{"stor-mdadm", "STOR", "mdadm", "Linux Kernel Software RAID Manager"},
{"stor-lvm2", "STOR", "lvm2", "Logical Volume & Thin Provisioning"},
{"stor-borg", "STOR", "borgbackup", "Deduplicated Encrypted Backups"},
{"mon-btop", "MON", "btop", "Real-Time Hardware Telemetry TUI"},
{"mon-sysstat", "MON", "sysstat", "iostat, mpstat & sar Kernel Counters"},
{"mon-bpftrace", "MON", "bpftrace", "Dynamic Kernel eBPF Tracing CLI"},
{"mon-fio", "MON", "fio", "Enterprise NVMe IOPS Stress Tester"},
{"mon-prom", "MON", "prometheus-node-exporter", "Cluster Metrics Exporter"},
{"mon-chrony", "MON", "chrony", "Sub-Millisecond NTP Clock Sync"},
}

var BundlePresets = map[string][]string{
"game-host":   {"net-ethtool", "net-iperf3", "net-tc", "sec-nft", "sec-ipset", "sec-f2b", "hw-cpupower", "hw-numa", "hw-sensors", "mon-btop", "mon-sysstat", "mon-chrony"},
"vps-node":    {"virt-kvm", "virt-libvirt", "virt-cloudinit", "net-wg", "sec-nft", "sec-audit", "stor-zfs", "stor-lvm2", "hw-smart", "hw-nvme", "mon-prom"},
"sec-core":    {"sec-nft", "sec-ipset", "sec-f2b", "sec-audit", "sec-lynis", "sec-aide", "sec-rkhunter", "net-tcpdump"},
"net-edge":    {"net-haproxy", "net-nginx", "net-bird2", "net-frr", "net-unbound", "net-wg", "net-ethtool", "net-tc"},
"storage-pro": {"stor-zfs", "stor-mdadm", "stor-lvm2", "stor-borg", "hw-smart", "hw-nvme", "mon-fio"},
}

func detectPkgManager() string {
for _, m := range []string{"apt-get", "dnf", "yum", "pacman", "apk"} {
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

var arsenalCmd = &cobra.Command{Use: "arsenal", Short: "Manage & deploy 42+ bare-metal hosting modules"}

var arsenalListCmd = &cobra.Command{
Use: "list", Short: "List indexed modules (--cat=NET|SEC|VIRT|HW|STOR|MON)",
Run: func(cmd *cobra.Command, args []string) {
cat, _ := cmd.Flags().GetString("cat")
cat = strings.ToUpper(cat)
PrintBanner()
fmt.Printf("[%sINF%s] Modules: %d | Bundles: %d | PkgMgr: %s\n\n", Cyan, Reset, len(ArsenalRegistry), len(BundlePresets), detectPkgManager())
for _, t := range ArsenalRegistry {
if cat != "" && t.Category != cat {
continue
}
fmt.Printf("%s%-14s%s %-5s %-24s %s\n", Bold, t.ID, Reset, t.Category, t.PackageName, t.Role)
}
},
}

var arsenalBundleCmd = &cobra.Command{
Use: "bundle [preset]", Short: "Deploy hosting stack (game-host|vps-node|sec-core|net-edge|storage-pro)", Args: cobra.ExactArgs(1),
Run: func(cmd *cobra.Command, args []string) {
preset := strings.ToLower(args)
PrintBanner()
mods, ok := BundlePresets[preset]
if !ok {
fmt.Printf("[%sERR%s] Unknown bundle: %s\n", Red+Bold, Reset, preset)
return
}
fmt.Printf("[%sBUNDLE%s] Stack [%s] (%d modules):\n\n", Cyan+Bold, Reset, strings.ToUpper(preset), len(mods))
for i, id := range mods {
if t, found := resolvePackage(id); found {
fmt.Printf("[%02d/%02d] %-14s -> %s\n", i+1, len(mods), t.ID, t.PackageName)
}
}
},
}

func init() {
arsenalListCmd.Flags().String("cat", "", "Filter by category")
arsenalCmd.AddCommand(arsenalListCmd, arsenalBundleCmd)
rootCmd.AddCommand(arsenalCmd)
}
