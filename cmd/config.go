package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

const defaultConfig = `# RAWOPS declarative host configuration
[node]
hostname = "raw-node-01"
role = "game-host"

[performance]
profile = "game"
cpu_governor = "performance"
tcp_congestion = "bbr"
vm_swappiness = 1

[shield]
mode = "strict"
allowed_ports = "22,25565,19132"
enable_syn_cookie = true
`

type HostConfig struct {
	Hostname     string
	Role         string
	Profile      string
	Governor     string
	Congestion   string
	Swappiness   int
	ShieldMode   string
	AllowedPorts []string
	Syncookies   bool
}

// parseConfig intentionally supports the small, documented TOML subset emitted by init-config.
// Unknown keys are rejected so typos cannot silently become production policy.
func parseConfig(data string) (HostConfig, error) {
	var c HostConfig
	section := ""
	seen := map[string]bool{}
	scanner := bufio.NewScanner(strings.NewReader(data))
	for lineNo := 1; scanner.Scan(); lineNo++ {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSpace(line[1 : len(line)-1])
			if section != "node" && section != "performance" && section != "shield" {
				return c, fmt.Errorf("line %d: unsupported section %q", lineNo, section)
			}
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 || section == "" {
			return c, fmt.Errorf("line %d: expected key = value inside a section", lineNo)
		}
		key, raw := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
		full := section + "." + key
		if seen[full] {
			return c, fmt.Errorf("line %d: duplicate key %s", lineNo, full)
		}
		seen[full] = true
		value := strings.Trim(raw, `"`)
		switch full {
		case "node.hostname":
			c.Hostname = value
		case "node.role":
			c.Role = value
		case "performance.profile":
			c.Profile = value
		case "performance.cpu_governor":
			c.Governor = value
		case "performance.tcp_congestion":
			c.Congestion = value
		case "performance.vm_swappiness":
			n, err := strconv.Atoi(raw)
			if err != nil {
				return c, fmt.Errorf("line %d: invalid integer", lineNo)
			}
			c.Swappiness = n
		case "shield.mode":
			c.ShieldMode = value
		case "shield.allowed_ports":
			for _, p := range strings.Split(value, ",") {
				p = strings.TrimSpace(p)
				if p != "" {
					c.AllowedPorts = append(c.AllowedPorts, p)
				}
			}
		case "shield.enable_syn_cookie":
			b, err := strconv.ParseBool(raw)
			if err != nil {
				return c, fmt.Errorf("line %d: invalid boolean", lineNo)
			}
			c.Syncookies = b
		default:
			return c, fmt.Errorf("line %d: unknown key %s", lineNo, full)
		}
	}
	if err := scanner.Err(); err != nil {
		return c, err
	}
	if c.Hostname == "" {
		return c, fmt.Errorf("node.hostname is required")
	}
	if c.Role != "game-host" && c.Role != "vps-node" && c.Role != "sec-core" {
		return c, fmt.Errorf("unsupported node.role %q", c.Role)
	}
	if c.Profile != "game" && c.Profile != "vps" {
		return c, fmt.Errorf("performance.profile must be game or vps")
	}
	if c.Governor != "performance" && c.Governor != "ondemand" && c.Governor != "schedutil" {
		return c, fmt.Errorf("unsupported cpu_governor %q", c.Governor)
	}
	if c.Congestion != "bbr" && c.Congestion != "cubic" && c.Congestion != "reno" {
		return c, fmt.Errorf("unsupported tcp_congestion %q", c.Congestion)
	}
	if c.Swappiness < 0 || c.Swappiness > 100 {
		return c, fmt.Errorf("vm_swappiness must be between 0 and 100")
	}
	if c.ShieldMode != "strict" && c.ShieldMode != "lockdown" {
		return c, fmt.Errorf("shield.mode must be strict or lockdown")
	}
	if len(c.AllowedPorts) == 0 {
		return c, fmt.Errorf("shield.allowed_ports must contain at least one port")
	}
	for _, p := range c.AllowedPorts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 1 || n > 65535 {
			return c, fmt.Errorf("invalid port %q", p)
		}
	}
	return c, nil
}

var initConfigCmd = &cobra.Command{
	Use:   "init-config",
	Short: "Create a starter rawops.toml without overwriting an existing file",
	RunE: func(cmd *cobra.Command, args []string) error {
		path, _ := cmd.Flags().GetString("output")
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("%s already exists; refusing to overwrite", path)
		}
		if err := os.WriteFile(path, []byte(defaultConfig), 0600); err != nil {
			return err
		}
		fmt.Printf("Created %s (mode 0600)\n", path)
		return nil
	},
}

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Validate a configuration and show its execution plan (no changes by default)",
	RunE: func(cmd *cobra.Command, args []string) error {
		path, _ := cmd.Flags().GetString("config")
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read config: %w", err)
		}
		cfg, err := parseConfig(string(data))
		if err != nil {
			return fmt.Errorf("configuration invalid: %w", err)
		}
		fmt.Println("Configuration valid.")
		fmt.Printf("Node: %s | role: %s\n", cfg.Hostname, cfg.Role)
		fmt.Printf("Performance: profile=%s governor=%s congestion=%s swappiness=%d\n", cfg.Profile, cfg.Governor, cfg.Congestion, cfg.Swappiness)
		fmt.Printf("Shield: mode=%s allowed_ports=%s syncookies=%t\n", cfg.ShieldMode, strings.Join(cfg.AllowedPorts, ","), cfg.Syncookies)
		fmt.Println("PLAN ONLY: this command has not modified the host. Live policy application is intentionally gated pending the transactional executor.")
		return nil
	},
}

func init() {
	initConfigCmd.Flags().String("output", "rawops.toml", "Output configuration path")
	applyCmd.Flags().String("config", "rawops.toml", "Configuration file")
	rootCmd.AddCommand(initConfigCmd, applyCmd)
}
