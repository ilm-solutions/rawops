package cmd

import (
"fmt"
"os"
"runtime"
"strconv"
"strings"
"time"

"github.com/spf13/cobra"
)

func renderBar(percent float64, width int) string {
if percent < 0 {
percent = 0
}
if percent > 100 {
percent = 100
}
filled := int((percent / 100.0) * float64(width))
return "[" + strings.Repeat("#", filled) + strings.Repeat("-", width-filled) + "]"
}

func getMemStats() (totalMB, usedMB uint64, usedPct float64) {
data, err := os.ReadFile("/proc/meminfo")
if err != nil {
return 0, 0, 0
}
var memTotal, memAvail uint64
for _, line := range strings.Split(string(data), "\n") {
fields := strings.Fields(line)
if len(fields) < 2 {
continue
}
val, _ := strconv.ParseUint(fields, 10, 64)
if fields == "MemTotal:" {
memTotal = val / 1024
} else if fields == "MemAvailable:" {
memAvail = val / 1024
}
}
if memTotal > 0 {
used := memTotal - memAvail
return memTotal, used, (float64(used) / float64(memTotal)) * 100.0
}
return 0, 0, 0
}

func getNetPackets() (rxPkts, txPkts uint64) {
data, err := os.ReadFile("/proc/net/dev")
if err != nil {
return 0, 0
}
for _, line := range strings.Split(string(data), "\n") {
if !strings.Contains(line, ":") || strings.Contains(line, "lo:") {
continue
}
parts := strings.Split(line, ":")
fields := strings.Fields(parts)
if len(fields) >= 10 {
rx, _ := strconv.ParseUint(fields, 10, 64)
tx, _ := strconv.ParseUint(fields, 10, 64)
rxPkts += rx
txPkts += tx
}
}
return rxPkts, txPkts
}

var monitorCmd = &cobra.Command{
Use:   "monitor",
Short: "Real-time CLI hardware, memory, and network PPS telemetry",
Run: func(cmd *cobra.Command, args []string) {
ticks, _ := cmd.Flags().GetInt("ticks")
PrintBanner()
prevRx, prevTx := getNetPackets()
time.Sleep(500 * time.Millisecond)

for i := 1; i <= ticks; i++ {
totalMB, usedMB, memPct := getMemStats()
curRx, curTx := getNetPackets()
rxPPS := curRx - prevRx
txPPS := curTx - prevTx
prevRx, prevTx = curRx, curTx

fmt.Printf("\033[H\033[2J")
PrintBanner()
fmt.Printf("[%sSAMPLE %02d/%02d%s] ARCH: %s | CORES: %d | SILICON: %s\n", Bold, i, ticks, Reset, runtime.GOARCH, runtime.NumCPU(), GetSiliconVendor())
fmt.Println("------------------------------------------------------------------------")
fmt.Printf("MEMORY : %s %6.2f%% (%d MB / %d MB)\n", renderBar(memPct, 24), memPct, usedMB, totalMB)
fmt.Printf("NET_IO : RX_PPS: %d pkt/s | TX_PPS: %d pkt/s\n", rxPPS, txPPS)
fmt.Printf("CPU_GOV: %s\n", readKernelParam("/sys/devices/system/cpu/cpu0/cpufreq/scaling_governor"))
fmt.Println("------------------------------------------------------------------------")
time.Sleep(1 * time.Second)
}
fmt.Printf("[%sOK%s] Telemetry capture complete.\n", Green+Bold, Reset)
},
}

func init() {
monitorCmd.Flags().Int("ticks", 5, "Number of 1-second telemetry refresh cycles")
rootCmd.AddCommand(monitorCmd)
}
