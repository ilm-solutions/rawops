package cmd

import (
	"fmt"
	"strconv"
	"strings"
)

// parseShieldPorts validates and normalizes a comma-separated
// list of TCP/UDP port numbers.
//
// It preserves input order and removes duplicate ports.
// It performs no filesystem or host-system operations.
func parseShieldPorts(raw string) ([]string, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, fmt.Errorf("at least one port is required")
	}

	parts := strings.Split(raw, ",")
	ports := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)

		if part == "" {
			return nil, fmt.Errorf("empty port entry")
		}

		n, err := strconv.Atoi(part)
		if err != nil || n < 1 || n > 65535 {
			return nil, fmt.Errorf("invalid port %q", part)
		}

		normalized := strconv.Itoa(n)

		if _, exists := seen[normalized]; exists {
			continue
		}

		seen[normalized] = struct{}{}
		ports = append(ports, normalized)
	}

	return ports, nil
}
