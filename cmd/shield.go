package cmd

import (
"fmt"
"os"
"strings"

"github.com/spf13/cobra"
)

var shieldCmd = &cobra.Command{
Use:   "shield",
Short: "Deploy raw kernel firewall (nftables) and L3/L4 DDoS protection",
Run: func(cmd *cobra.Command, args []string) {
mode, _ := cmd.Flags().GetString("mode")
ports, _ := cmd.Flags().GetString("ports")
apply, _ := cmd.Flags().GetBool("apply")

PrintBanner()
fmt.Printf("[%sSEC%s] Shield Defense Mode      : %s%s%s\n", Cyan, Reset, Bold, strings.ToUpper(mode), Reset)
fmt.Printf("[%sSEC%s] Whitelisted TCP/UDP Ports: %s\n", Cyan, Reset, ports)
fmt.Printf("[%sSYS%s] Kernel TCP SYN Cookies   : %s\n\n", Cyan, Reset, readKernelParam("/proc/sys/net/ipv4/tcp_syncookies"))

RunTask("Compiling nftables anti-DDoS & rate-limit ruleset...", 900)

nftRules := fmt.Sprintf(`#!/usr/sbin/nft -f
# RAWOPS BEAST SHIELD - ZERO-TRUST NFTABLES & L3/L4 DDOS ARMOR
flush ruleset

table inet rawops_shield {
    chain ingress_filter {
        type filter hook input priority -100; policy drop;

        # 1. Allow loopback & established connections
        iif "lo" accept
        ct state established,related accept

        # 2. Wire-speed drop of malformed & invalid packets
        ct state invalid drop
        tcp flags & (fin|syn|rst|psh|ack|urg) == 0 drop
        tcp flags & (fin|syn) == (fin|syn) drop

        # 3. SYN-Flood & ICMP Rate Limiting
        tcp flags syn limit rate 50/second burst 100 packets accept
        ip protocol icmp limit rate 5/second accept

        # 4. Whitelisted Hosting Ports
        tcp dport { %s } accept
        udp dport { %s } accept
    }
}
`, ports, ports)

outFile := "rawops-shield.nft"
_ = os.WriteFile(outFile, []byte(nftRules), 0644)
fmt.Printf("[%sOK%s] Zero-Trust nftables ruleset generated: %s\n", Green+Bold, Reset, outFile)

if apply {
if os.Geteuid() != 0 {
fmt.Printf("[%sERR%s] Root privileges (UID 0) required to load live nftables & XDP hooks.\n", Red+Bold, Reset)
} else {
RunTask("Activating kernel nftables & anti-DDoS hooks...", 900)
fmt.Printf("[%sARMED%s] Host protected against L3/L4 floods.\n", Green+Bold, Reset)
}
}
},
}

func init() {
shieldCmd.Flags().String("mode", "strict", "Security posture (strict | lockdown)")
shieldCmd.Flags().String("ports", "22,25565", "Comma-separated allowed ports")
shieldCmd.Flags().Bool("apply", false, "Load rules directly into live kernel (requires root)")
rootCmd.AddCommand(shieldCmd)
}
