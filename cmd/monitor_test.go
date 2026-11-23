package cmd

import (
	"testing"
	"time"
)

func TestPacketRate(t *testing.T) {
	tests := []struct {
		name      string
		cur, prev uint64
		elapsed   time.Duration
		want      float64
	}{
		{name: "one second", cur: 150, prev: 50, elapsed: time.Second, want: 100},
		{name: "half second", cur: 150, prev: 50, elapsed: 500 * time.Millisecond, want: 200},
		{name: "counter reset", cur: 2, prev: 50, elapsed: time.Second, want: 0},
		{name: "zero interval", cur: 150, prev: 50, elapsed: 0, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := packetRate(tt.cur, tt.prev, tt.elapsed); got != tt.want {
				t.Fatalf("packetRate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRenderBarClampsWidthAndPercent(t *testing.T) {
	if got := renderBar(100, 4); got != "[####]" {
		t.Fatalf("renderBar() = %q", got)
	}
	if got := renderBar(-10, 4); got != "[----]" {
		t.Fatalf("renderBar() = %q", got)
	}
	if got := renderBar(50, -2); got != "[]" {
		t.Fatalf("renderBar() negative width = %q", got)
	}
}

func TestParseCPUStatAndUsage(t *testing.T) {
	previous, err := parseCPUStat([]byte("cpu 100 0 50 800 50 0 0 0 0 0\ncpu0 20 0 10 200 10 0 0 0 0 0\n"))
	if err != nil {
		t.Fatal(err)
	}
	current, err := parseCPUStat([]byte("cpu 120 0 75 850 55 0 0 0 0 0\n"))
	if err != nil {
		t.Fatal(err)
	}
	got, ok := cpuUsage(previous, current)
	if !ok || got != 45 {
		t.Fatalf("cpuUsage() = %v, %v; want 45, true", got, ok)
	}
}

func TestCPUStatRejectsMalformedAndResetCounters(t *testing.T) {
	if _, err := parseCPUStat([]byte("not-cpu 1 2 3 4\n")); err == nil {
		t.Fatal("expected malformed aggregate line to fail")
	}
	if _, err := parseCPUStat([]byte("cpu 1 2 bad 4 5\n")); err == nil {
		t.Fatal("expected invalid counter to fail")
	}
	if _, ok := cpuUsage(cpuCounters{total: 100, idle: 80}, cpuCounters{total: 90, idle: 70}); ok {
		t.Fatal("expected reset counters to be rejected")
	}
}

func TestFormatCPUUsage(t *testing.T) {
	if got := formatCPUUsage(12.34, true); got != "12.3%" {
		t.Fatalf("formatCPUUsage() = %q", got)
	}
	if got := formatCPUUsage(0, false); got != "N/A (counter unavailable)" {
		t.Fatalf("formatCPUUsage() unavailable = %q", got)
	}
}

func TestParseCPUStatDoesNotDoubleCountGuestTime(t *testing.T) {
	counters, err := parseCPUStat([]byte("cpu 100 0 50 800 50 0 0 0 500 200\n"))
	if err != nil {
		t.Fatal(err)
	}
	if counters.total != 1000 {
		t.Fatalf("total = %d, want 1000 (guest fields are already included)", counters.total)
	}
}
