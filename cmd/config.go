package cmd

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
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

var hostnamePattern = regexp.MustCompile(`(?i)^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?(?:\.[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?)*$`)

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
	sections := map[string]bool{}
	scanner := bufio.NewScanner(strings.NewReader(data))
	scanner.Buffer(make([]byte, 4096), 1024*1024)

	parseString := func(raw string, lineNo int) (string, error) {
		value, err := strconv.Unquote(raw)
		if err != nil {
			return "", fmt.Errorf("line %d: expected a valid quoted string", lineNo)
		}
		if strings.TrimSpace(value) == "" {
			return "", fmt.Errorf("line %d: value must not be empty", lineNo)
		}
		return value, nil
	}

	for lineNo := 1; scanner.Scan(); lineNo++ {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") || strings.HasSuffix(line, "]") {
			if !strings.HasPrefix(line, "[") || !strings.HasSuffix(line, "]") || strings.Count(line, "[") != 1 || strings.Count(line, "]") != 1 {
				return c, fmt.Errorf("line %d: malformed section header", lineNo)
			}
			section = strings.TrimSpace(line[1 : len(line)-1])
			if section != "node" && section != "performance" && section != "shield" {
				return c, fmt.Errorf("line %d: unsupported section %q", lineNo, section)
			}
			if sections[section] {
				return c, fmt.Errorf("line %d: duplicate section %q", lineNo, section)
			}
			sections[section] = true
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 || section == "" {
			return c, fmt.Errorf("line %d: expected key = value inside a section", lineNo)
		}
		key, raw := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
		if key == "" || raw == "" || strings.ContainsAny(key, " \t") {
			return c, fmt.Errorf("line %d: malformed key or value", lineNo)
		}
		full := section + "." + key
		if seen[full] {
			return c, fmt.Errorf("line %d: duplicate key %s", lineNo, full)
		}
		seen[full] = true

		switch full {
		case "node.hostname", "node.role", "performance.profile", "performance.cpu_governor", "performance.tcp_congestion", "shield.mode", "shield.allowed_ports":
			value, err := parseString(raw, lineNo)
			if err != nil {
				return c, err
			}
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
			case "shield.mode":
				c.ShieldMode = value
			case "shield.allowed_ports":
				ports, err := parseShieldPorts(value)
				if err != nil {
					return c, fmt.Errorf("line %d: invalid shield.allowed_ports: %w", lineNo, err)
				}
				c.AllowedPorts = ports
			}
		case "performance.vm_swappiness":
			n, err := strconv.Atoi(raw)
			if err != nil {
				return c, fmt.Errorf("line %d: invalid integer", lineNo)
			}
			c.Swappiness = n
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

	required := []string{
		"node.hostname", "node.role", "performance.profile", "performance.cpu_governor",
		"performance.tcp_congestion", "performance.vm_swappiness", "shield.mode",
		"shield.allowed_ports", "shield.enable_syn_cookie",
	}
	for _, key := range required {
		if !seen[key] {
			return c, fmt.Errorf("required configuration key %s is missing", key)
		}
	}
	if len(c.Hostname) > 253 || !hostnamePattern.MatchString(c.Hostname) {
		return c, fmt.Errorf("node.hostname must be a valid hostname label or FQDN")
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
	return c, nil
}

var initConfigCmd = &cobra.Command{
	Use:   "init-config",
	Short: "Create a starter rawops.toml without overwriting an existing file",
	RunE: func(cmd *cobra.Command, args []string) error {
		path, _ := cmd.Flags().GetString("output")
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			if os.IsExist(err) {
				return fmt.Errorf("%s already exists; refusing to overwrite", path)
			}
			return err
		}
		if _, err := f.WriteString(defaultConfig); err != nil {
			f.Close()
			os.Remove(path)
			return err
		}
		if err := f.Sync(); err != nil {
			f.Close()
			os.Remove(path)
			return err
		}
		if err := f.Close(); err != nil {
			os.Remove(path)
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
