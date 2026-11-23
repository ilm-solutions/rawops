package cmd

import (
	"fmt"
	"time"
)

const (
	Reset  = "\033[0m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Cyan   = "\033[36m"
	Bold   = "\033[1m"
)

func PrintBanner() {
	fmt.Println(Cyan + Bold + "========================================================================")
	fmt.Printf(" RAWOPS ENGINE v%s | BARE-METAL CONTROL PLANE [CLI-ONLY]\n", Version)
	fmt.Println("========================================================================" + Reset)
}

func RunTask(label string, durationMs int) {
	frames := []string{"|", "/", "-", "\\"}
	steps := durationMs / 80
	for i := 0; i < steps; i++ {
		fmt.Printf("\r[%s*%s] %-36s [%s]", Yellow, Reset, label, frames[i%len(frames)])
		time.Sleep(80 * time.Millisecond)
	}
	fmt.Printf("\r[%s*%s] %-36s [%s OK %s]\n", Green, Reset, label, Green+Bold, Reset)
}
