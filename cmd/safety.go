package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const BackupDir = "/var/backups/rawops"

func atomicWrite(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".rawops-*.tmp")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if err = f.Chmod(mode); err == nil {
		_, err = f.Write(data)
	}
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// CreateHostSnapshot records RAWOPS-owned state only. It never snapshots or flushes
// the complete host firewall. A snapshot is mandatory before a live policy change.
func CreateHostSnapshot() error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("root privileges required to snapshot host state")
	}
	if err := os.MkdirAll(BackupDir, 0700); err != nil {
		return err
	}
	stamp := time.Now().UTC().Format("20060102T150405Z")
	dir := filepath.Join(BackupDir, stamp)
	if err := os.Mkdir(dir, 0700); err != nil {
		return err
	}

	sysctlData, err := os.ReadFile("/etc/sysctl.d/99-rawops.conf")
	if err == nil {
		if err := atomicWrite(filepath.Join(dir, "sysctl.conf"), sysctlData, 0600); err != nil {
			return err
		}
	} else if os.IsNotExist(err) {
		if err := atomicWrite(filepath.Join(dir, "NO_SYSCTL_CONFIG"), []byte(""), 0600); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("snapshot sysctl config: %w", err)
	}

	if _, err := exec.LookPath("nft"); err == nil {
		live, listErr := runCommand(10*time.Second, "nft", "list", "table", "inet", "rawops_shield")
		if listErr == nil {
			if err := atomicWrite(filepath.Join(dir, "shield-live.nft"), live, 0600); err != nil {
				return err
			}
		} else {
			if err := atomicWrite(filepath.Join(dir, "NO_RAWOPS_TABLE"), []byte(""), 0600); err != nil {
				return err
			}
		}
	} else {
		if err := atomicWrite(filepath.Join(dir, "NO_RAWOPS_TABLE"), []byte("nft unavailable"), 0600); err != nil {
			return err
		}
	}

	if err := atomicWrite(filepath.Join(BackupDir, "LATEST"), []byte(stamp+"\n"), 0600); err != nil {
		return err
	}
	return nil
}

var rollbackCmd = &cobra.Command{
	Use:   "rollback",
	Short: "Restore the latest RAWOPS-owned configuration snapshot",
	RunE: func(cmd *cobra.Command, args []string) error {
		if os.Geteuid() != 0 {
			return fmt.Errorf("rollback requires root")
		}
		stampBytes, err := os.ReadFile(filepath.Join(BackupDir, "LATEST"))
		if err != nil {
			return fmt.Errorf("no RAWOPS snapshot found: %w", err)
		}
		dir := filepath.Join(BackupDir, strings.TrimSpace(string(stampBytes)))

		sysctlPath := "/etc/sysctl.d/99-rawops.conf"
		if data, err := os.ReadFile(filepath.Join(dir, "sysctl.conf")); err == nil {
			if err := atomicWrite(sysctlPath, data, 0644); err != nil {
				return err
			}
			if out, err := runCommand(15*time.Second, "sysctl", "-p", sysctlPath); err != nil {
				return fmt.Errorf("restored sysctl file but reload failed: %w: %s", err, strings.TrimSpace(string(out)))
			}
		} else if _, markerErr := os.Stat(filepath.Join(dir, "NO_SYSCTL_CONFIG")); markerErr == nil {
			if err := os.Remove(sysctlPath); err != nil && !os.IsNotExist(err) {
				return err
			}
		} else {
			return fmt.Errorf("snapshot has no sysctl state")
		}

		if _, err := exec.LookPath("nft"); err == nil {
			if _, listErr := runCommand(10*time.Second, "nft", "list", "table", "inet", "rawops_shield"); listErr == nil {
				if out, delErr := runCommand(10*time.Second, "nft", "delete", "table", "inet", "rawops_shield"); delErr != nil {
					return fmt.Errorf("could not remove current RAWOPS table: %w: %s", delErr, strings.TrimSpace(string(out)))
				}
			}
			if data, readErr := os.ReadFile(filepath.Join(dir, "shield-live.nft")); readErr == nil {
				restoreFile := filepath.Join(dir, "restore.nft")
				if err := atomicWrite(restoreFile, data, 0600); err != nil {
					return err
				}
				if out, applyErr := runCommand(15*time.Second, "nft", "-f", restoreFile); applyErr != nil {
					return fmt.Errorf("firewall restore failed: %w: %s", applyErr, strings.TrimSpace(string(out)))
				}
			} else if _, markerErr := os.Stat(filepath.Join(dir, "NO_RAWOPS_TABLE")); markerErr != nil {
				return fmt.Errorf("snapshot has no firewall state")
			}
		}
		fmt.Printf("Restored RAWOPS-owned state from snapshot %s. Non-RAWOPS firewall tables were not modified.\n", strings.TrimSpace(string(stampBytes)))
		return nil
	},
}

func init() { rootCmd.AddCommand(rollbackCmd) }
