package cmd

import (
"fmt"
"os"
"os/exec"
"strings"

"github.com/spf13/cobra"
)

var shieldCmd = &cobra.Command{
Use:   "shield",
Short: "Deploy live nftables zero-trust firewall and L3/L4 DDoS armor",
Run: func(cmd *cobra.Command, args []string) {
mode, _ := cmd.Flags().GetString("mode")
ports, _ := cmd.Flags().GetString("ports")
apply, _ := cmd.Flags().GetBool("apply")

PrintBanner()
fmt.Printf("[%sSEC%s] Shield Posture           : %s%s%s\n", Cyan, Reset, Bold, strings.ToUpper(mode), Reset)
fmt.Printf("[%sSEC%s] Whitelisted TCP/UDP Ports: %s\n", Cyan, Reset, ports)
fmt.Printf("[%sSYS%s] Kernel TCP SYN Cookies   : %s\n\n", Cyan, Reset, readKernelParam("/proc/sys/net/ipv4/tcp_syncookies"))

RunTask("Compiling zero-trust nftables anti-DDoS ruleset...", 700)
nftRules := fmt.Sprintf(`#!/usr/sbin/nft -f
flush ruleset

table inet rawops_shield {
    chain ingress_filter {
        type filter hook input priority -100; policy drop;

        iif "lo" accept
        ct state established,related accept

        # Drop malformed & invalid packets at wire speed
        ct state invalid drop
        tcp flags & (fin|syn|rst|psh|ack|urg) == 0 drop
        tcp flags & (fin|syn) == (fin|syn) drop

        # SYN-Flood & ICMP Rate Limiting
        tcp flags syn limit rate 50/second burst 100 packets accept
        ip protocol icmp limit rate 5/second accept

        # Whitelisted Hosting Ports
        tcp dport { %s } accept
        udp dport { %s } accept
    }
}
`, ports, ports)

if !apply {
outFile := "rawops-shield.nft"
_ = os.WriteFile(outFile, []byte(nftRules), 0644)
fmt.Printf("[%sDRY-RUN%s] Ruleset saved: %s (Use --apply as root to enforce live)\n", Yellow+Bold, Reset, outFile)
return
}

if os.Geteuid() != 0 {
fmt.Printf("[%sERR%s] Root privileges (UID 0) required to enforce live nftables.\n", Red+Bold, Reset)
return
}

RunTask("Snapshotting existing firewall state...", 500)
_ = CreateHostSnapshot()
_ = os.MkdirAll("/etc/rawops", 0700)

nftPath := "/etc/rawops/shield.nft"
_ = os.WriteFile(nftPath, []byte(nftRules), 0600)
RunTask("Loading live nftables ruleset & SYN-cookies...", 800)
_ = exec.Command("sysctl", "-w", "net.ipv4.tcp_syncookies=1").Run()
_ = exec.Command("nft", "-f", nftPath).Run()

fmt.Printf("\n[%sARMED%s] Zero-Trust nftables DDoS shield active on host.\n", Green+Bold, Reset)
},
}

func init() {
shieldCmd.Flags().String("mode", "strict", "Security posture (strict | lockdown)")
shieldCmd.Flags().String("ports", "22,25565", "Comma-separated allowed ports")
shieldCmd.Flags().Bool("apply", false, "Enforce rules directly in Linux kernel (requires root)")
rootCmd.AddCommand(shieldCmd)
}
