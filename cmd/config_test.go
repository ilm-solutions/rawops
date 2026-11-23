package cmd

import (
	"strings"
	"testing"
)

func TestParseConfigDefault(t *testing.T) {
	cfg, err := parseConfig(defaultConfig)
	if err != nil {
		t.Fatalf("parse default config: %v", err)
	}
	if cfg.Hostname != "raw-node-01" || cfg.Profile != "game" || len(cfg.AllowedPorts) != 3 {
		t.Fatalf("unexpected config: %#v", cfg)
	}
}

func TestParseConfigRejectsUnknownKey(t *testing.T) {
	bad := strings.Replace(defaultConfig, "[shield]", "unsafe_magic = true\n\n[shield]", 1)
	if _, err := parseConfig(bad); err == nil {
		t.Fatal("expected unknown key to fail")
	}
}

func TestParseConfigRejectsInvalidPort(t *testing.T) {
	bad := "[node]\nhostname = \"node\"\nrole = \"game-host\"\n" +
		"[performance]\nprofile = \"game\"\ncpu_governor = \"performance\"\n" +
		"tcp_congestion = \"bbr\"\nvm_swappiness = 1\n" +
		"[shield]\nmode = \"strict\"\nallowed_ports = \"99999\"\nenable_syn_cookie = true\n"
	if _, err := parseConfig(bad); err == nil {
		t.Fatal("expected invalid port to fail")
	}
}

func TestParseConfigRejectsMissingRequiredKey(t *testing.T) {
	bad := `[node]
hostname = "node-1"
role = "game-host"
[performance]
profile = "game"
cpu_governor = "performance"
tcp_congestion = "bbr"
vm_swappiness = 1
[shield]
mode = "strict"
allowed_ports = "22"
`
	if _, err := parseConfig(bad); err == nil {
		t.Fatal("expected missing required key to fail")
	}
}

func TestParseConfigRejectsMalformedQuotedString(t *testing.T) {
	bad := strings.Replace(defaultConfig, `hostname = "raw-node-01"`, `hostname = "raw-node-01`, 1)
	if _, err := parseConfig(bad); err == nil {
		t.Fatal("expected malformed quoted string to fail")
	}
}

func TestParseConfigRejectsInvalidHostname(t *testing.T) {
	bad := strings.Replace(defaultConfig, `hostname = "raw-node-01"`, `hostname = "bad hostname"`, 1)
	if _, err := parseConfig(bad); err == nil {
		t.Fatal("expected invalid hostname to fail")
	}
}

func TestParseConfigRejectsEmptyPortEntry(t *testing.T) {
	bad := strings.Replace(defaultConfig, `allowed_ports = "22,25565,19132"`, `allowed_ports = "22,,19132"`, 1)
	if _, err := parseConfig(bad); err == nil {
		t.Fatal("expected empty port entry to fail")
	}
}

func TestParseConfigRejectsDuplicateSection(t *testing.T) {
	bad := defaultConfig + "\n[node]\nhostname = \"another-node\"\n"
	if _, err := parseConfig(bad); err == nil {
		t.Fatal("expected duplicate section to fail")
	}
}
