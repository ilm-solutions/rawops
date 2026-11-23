package cmd

import "testing"

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
	bad := defaultConfig + "\n[performance]\nunsafe_magic = true\n"
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
