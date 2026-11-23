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

type cpuCounters struct{ total, idle uint64 }

func parseCPUStat(data []byte) (cpuCounters, error) {
	line := strings.SplitN(string(data), "\n", 2)[0]
	fields := strings.Fields(line)
	if len(fields) < 5 || fields[0] != "cpu" {
		return cpuCounters{}, fmt.Errorf("aggregate CPU counters unavailable")
	}
	var c cpuCounters
	// Linux guest and guest_nice counters are already included in user/nice;
	// exclude those final two fields to avoid double-counting total CPU time.
	counterFields := fields[1:]
	if len(counterFields) > 8 {
		counterFields = counterFields[:8]
	}
	values := make([]uint64, len(counterFields))
	for i, field := range counterFields {
		value, err := strconv.ParseUint(field, 10, 64)
		if err != nil {
			return cpuCounters{}, fmt.Errorf("invalid CPU counter %q: %w", field, err)
		}
		values[i] = value
		c.total += value
	}
	c.idle = values[3]
	if len(values) > 4 {
		c.idle += values[4]
	}
	return c, nil
}

func readCPUStat() (cpuCounters, error) {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return cpuCounters{}, err
	}
	return parseCPUStat(data)
}

func cpuUsage(previous, current cpuCounters) (float64, bool) {
	if current.total <= previous.total || current.idle < previous.idle {
		return 0, false
	}
	totalDelta, idleDelta := current.total-previous.total, current.idle-previous.idle
	if idleDelta > totalDelta {
		return 0, false
	}
	return 100 * float64(totalDelta-idleDelta) / float64(totalDelta), true
}

func formatCPUUsage(percent float64, valid bool) string {
	if !valid {
		return "N/A (counter unavailable)"
	}
	return fmt.Sprintf("%.1f%%", percent)
}

func renderBar(percent float64, width int) string {
	if width < 0 {
		width = 0
	}
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
		f := strings.Fields(line)
		if len(f) < 2 {
			continue
		}
		val, err := strconv.ParseUint(f[1], 10, 64)
		if err != nil {
			continue
		}
		switch f[0] {
		case "MemTotal:":
			memTotal = val / 1024
		case "MemAvailable:":
			memAvail = val / 1024
		}
	}
	if memTotal == 0 {
		return 0, 0, 0
	}
	if memAvail > memTotal {
		memAvail = memTotal
	}
	used := memTotal - memAvail
	return memTotal, used, (float64(used) / float64(memTotal)) * 100.0
}

func getNetPackets() (rxPkts, txPkts uint64) {
	data, err := os.ReadFile("/proc/net/dev")
	if err != nil {
		return 0, 0
	}
	for _, line := range strings.Split(string(data), "\n") {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 || strings.TrimSpace(parts[0]) == "lo" {
			continue
		}
		fields := strings.Fields(parts[1])
		if len(fields) < 10 {
			continue
		}
		rx, rxErr := strconv.ParseUint(fields[1], 10, 64)
		tx, txErr := strconv.ParseUint(fields[9], 10, 64)
		if rxErr == nil {
			rxPkts += rx
		}
		if txErr == nil {
			txPkts += tx
		}
	}
	return rxPkts, txPkts
}

// packetRate returns zero for a reset/decreased counter and normalizes by elapsed time.
func packetRate(current, previous uint64, elapsed time.Duration) float64 {
	if elapsed <= 0 || current < previous {
		return 0
	}
	return float64(current-previous) / elapsed.Seconds()
}

var monitorCmd = &cobra.Command{
	Use:   "monitor",
	Short: "Display CPU, memory, and aggregate network packet telemetry",
	RunE: func(cmd *cobra.Command, args []string) error {
		ticks, _ := cmd.Flags().GetInt("ticks")
		if ticks < 1 || ticks > 3600 {
			return fmt.Errorf("--ticks must be between 1 and 3600")
		}
		PrintBanner()
		prevRx, prevTx := getNetPackets()
		prevCPU, prevCPUErr := readCPUStat()
		lastSample := time.Now()

		for i := 1; i <= ticks; i++ {
			time.Sleep(time.Second)
			now := time.Now()
			elapsed := now.Sub(lastSample)
			totalMB, usedMB, memPct := getMemStats()
			curRx, curTx := getNetPackets()
			curCPU, curCPUErr := readCPUStat()
			cpuPct, cpuOK := cpuUsage(prevCPU, curCPU)
			if prevCPUErr != nil || curCPUErr != nil {
				cpuOK = false
			}
			rxPPS := packetRate(curRx, prevRx, elapsed)
			txPPS := packetRate(curTx, prevTx, elapsed)
			prevRx, prevTx = curRx, curTx
			prevCPU, prevCPUErr = curCPU, curCPUErr
			lastSample = now

			fmt.Printf("\033[H\033[2J")
			PrintBanner()
			fmt.Printf("[%sSAMPLE %02d/%02d%s] ARCH: %s | CORES: %d | SILICON: %s\n", Bold, i, ticks, Reset, runtime.GOARCH, runtime.NumCPU(), GetSiliconVendor())
			fmt.Println("------------------------------------------------------------------------")
			fmt.Printf("CPU    : %s\n", formatCPUUsage(cpuPct, cpuOK))
			fmt.Printf("MEMORY : %s %6.2f%% (%d MB / %d MB)\n", renderBar(memPct, 24), memPct, usedMB, totalMB)
			fmt.Printf("NET_IO : RX_PPS: %.1f pkt/s | TX_PPS: %.1f pkt/s\n", rxPPS, txPPS)
			fmt.Printf("CPU_GOV: %s\n", readKernelParam("/sys/devices/system/cpu/cpu0/cpufreq/scaling_governor"))
			fmt.Println("------------------------------------------------------------------------")
		}
		fmt.Printf("[%sOK%s] Telemetry capture complete.\n", Green+Bold, Reset)
		return nil
	},
}

func init() {
	monitorCmd.Flags().Int("ticks", 5, "Number of 1-second telemetry refresh cycles (1-3600)")
	rootCmd.AddCommand(monitorCmd)
}
