package cmd

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const BackupDir = "/var/backups/rawops"

const snapshotStampLayout = "20060102T150405Z"

// validateSnapshotStamp accepts only canonical UTC snapshot identifiers.
func validateSnapshotStamp(stamp string) error {
	parsed, err := time.Parse(snapshotStampLayout, stamp)
	if err != nil || parsed.UTC().Format(snapshotStampLayout) != stamp {
		return fmt.Errorf("invalid snapshot identifier")
	}
	return nil
}

// validateSnapshotState verifies that a snapshot contains exactly one
// representation of each RAWOPS-owned state.
func validateSnapshotState(dir string) error {
	dirInfo, err := os.Lstat(dir)
	if err != nil {
		return fmt.Errorf("inspect snapshot directory: %w", err)
	}
	if !dirInfo.IsDir() || dirInfo.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("snapshot path is not a real directory")
	}
	hasFile := func(name string) (bool, error) {
		info, err := os.Lstat(filepath.Join(dir, name))
		if os.IsNotExist(err) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		if !info.Mode().IsRegular() {
			return false, fmt.Errorf("snapshot entry %s is not a regular file", name)
		}
		return true, nil
	}

	sysctl, err := hasFile("sysctl.conf")
	if err != nil {
		return err
	}
	noSysctl, err := hasFile("NO_SYSCTL_CONFIG")
	if err != nil {
		return err
	}
	if sysctl == noSysctl {
		return fmt.Errorf("snapshot must contain exactly one sysctl state")
	}

	shield, err := hasFile("shield-live.nft")
	if err != nil {
		return err
	}
	noShield, err := hasFile("NO_RAWOPS_TABLE")
	if err != nil {
		return err
	}
	if shield == noShield {
		return fmt.Errorf("snapshot must contain exactly one firewall state")
	}

	return nil
}

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
	if err == nil {
		err = f.Sync()
	}
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}

func writeTempFile(dir, pattern string, data []byte) (string, error) {
	f, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return "", err
	}
	path := f.Name()
	if err := f.Chmod(0600); err != nil {
		f.Close()
		os.Remove(path)
		return "", err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(path)
		return "", err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		os.Remove(path)
		return "", err
	}
	if err := f.Close(); err != nil {
		os.Remove(path)
		return "", err
	}
	return path, nil
}

// CreateHostSnapshot records RAWOPS-owned state only and returns the exact
// firewall-table inventory state captured by this snapshot.
func CreateHostSnapshot() (bool, error) {
	if os.Geteuid() != 0 {
		return false, fmt.Errorf("root privileges required to snapshot host state")
	}
	if _, err := exec.LookPath("nft"); err != nil {
		return false, fmt.Errorf("cannot snapshot RAWOPS firewall state: nftables is unavailable: %w", err)
	}
	inventory, err := runCommand(10*time.Second, "nft", "list", "tables")
	if err != nil {
		return false, fmt.Errorf("cannot inspect nftables table inventory: %w: %s", err, strings.TrimSpace(string(inventory)))
	}
	hasShieldTable, err := hasRawopsShieldTable(inventory)
	if err != nil {
		return false, fmt.Errorf("cannot interpret nftables table inventory: %w", err)
	}
	var shieldSnapshot []byte
	if hasShieldTable {
		shieldSnapshot, err = runCommand(10*time.Second, "nft", "list", "table", "inet", "rawops_shield")
		if err != nil {
			return false, fmt.Errorf("cannot snapshot RAWOPS firewall table: %w: %s", err, strings.TrimSpace(string(shieldSnapshot)))
		}
	}
	sysctlData, sysctlErr := os.ReadFile("/etc/sysctl.d/99-rawops.conf")
	if sysctlErr != nil && !os.IsNotExist(sysctlErr) {
		return false, fmt.Errorf("snapshot sysctl config: %w", sysctlErr)
	}
	if err := os.MkdirAll(BackupDir, 0700); err != nil {
		return false, err
	}
	backupInfo, err := os.Lstat(BackupDir)
	if err != nil {
		return false, err
	}
	if !backupInfo.IsDir() || backupInfo.Mode()&os.ModeSymlink != 0 {
		return false, fmt.Errorf("backup path is not a real directory")
	}
	if err := os.Chmod(BackupDir, 0700); err != nil {
		return false, err
	}
	stamp := time.Now().UTC().Format(snapshotStampLayout)
	dir := filepath.Join(BackupDir, stamp)
	if err := os.Mkdir(dir, 0700); err != nil {
		return false, err
	}
	complete := false
	defer func() {
		if !complete {
			_ = os.RemoveAll(dir)
		}
	}()
	if sysctlErr == nil {
		if err := atomicWrite(filepath.Join(dir, "sysctl.conf"), sysctlData, 0600); err != nil {
			return false, err
		}
	} else if err := atomicWrite(filepath.Join(dir, "NO_SYSCTL_CONFIG"), []byte(""), 0600); err != nil {
		return false, err
	}
	if hasShieldTable {
		if err := atomicWrite(filepath.Join(dir, "shield-live.nft"), shieldSnapshot, 0600); err != nil {
			return false, err
		}
	} else if err := atomicWrite(filepath.Join(dir, "NO_RAWOPS_TABLE"), []byte(""), 0600); err != nil {
		return false, err
	}
	manifest, err := buildSnapshotManifest(dir)
	if err != nil {
		return false, fmt.Errorf("build snapshot manifest: %w", err)
	}
	if err := atomicWrite(filepath.Join(dir, "SNAPSHOT.sha256"), manifest, 0600); err != nil {
		return false, fmt.Errorf("write snapshot manifest: %w", err)
	}
	// Keep a complete orphan snapshot if updating LATEST fails. Removing it
	// could leave LATEST pointing at a directory that no longer exists when a
	// directory sync reports an error after rename.
	complete = true
	if err := atomicWrite(filepath.Join(BackupDir, "LATEST"), []byte(stamp+"\n"), 0600); err != nil {
		return false, err
	}
	return hasShieldTable, nil
}

// buildFirewallRestoreBatch creates a single nftables transaction for restoring
// only the RAWOPS-owned table. It never flushes unrelated tables.
func buildFirewallRestoreBatch(currentTable, snapshotTable bool, snapshot []byte) []byte {
	var batch strings.Builder
	if currentTable {
		batch.WriteString("delete table inet rawops_shield\n")
	}
	if snapshotTable {
		batch.Write(snapshot)
		if len(snapshot) > 0 && snapshot[len(snapshot)-1] != '\n' {
			batch.WriteByte('\n')
		}
	}
	return []byte(batch.String())
}

var rollbackCmd = &cobra.Command{
	Use:   "rollback",
	Short: "Restore the latest RAWOPS-owned configuration snapshot",
	RunE: func(cmd *cobra.Command, args []string) error {
		if os.Geteuid() != 0 {
			return fmt.Errorf("rollback requires root")
		}
		if _, err := exec.LookPath("nft"); err != nil {
			return fmt.Errorf("rollback requires nftables; refusing partial restore: %w", err)
		}
		latestPath := filepath.Join(BackupDir, "LATEST")
		latestInfo, err := os.Lstat(latestPath)
		if err != nil {
			return fmt.Errorf("no RAWOPS snapshot found: %w", err)
		}
		if !latestInfo.Mode().IsRegular() {
			return fmt.Errorf("LATEST snapshot pointer is not a regular file")
		}
		stampBytes, err := os.ReadFile(latestPath)
		if err != nil {
			return fmt.Errorf("read LATEST snapshot pointer: %w", err)
		}
		stamp := strings.TrimSpace(string(stampBytes))
		if err := validateSnapshotStamp(stamp); err != nil {
			return fmt.Errorf("invalid LATEST snapshot identifier: %w", err)
		}
		dir := filepath.Join(BackupDir, stamp)
		if err := validateSnapshotState(dir); err != nil {
			return fmt.Errorf("snapshot validation failed: %w", err)
		}
		if err := validateSnapshotManifest(dir); err != nil {
			return fmt.Errorf("snapshot checksum validation failed: %w", err)
		}

		inventory, err := runCommand(10*time.Second, "nft", "list", "tables")
		if err != nil {
			return fmt.Errorf("cannot inspect current nftables state: %w: %s", err, strings.TrimSpace(string(inventory)))
		}
		currentTable, err := hasRawopsShieldTable(inventory)
		if err != nil {
			return fmt.Errorf("cannot interpret current nftables state: %w", err)
		}
		snapshotFirewall, err := os.ReadFile(filepath.Join(dir, "shield-live.nft"))
		snapshotHasTable := err == nil
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("read firewall snapshot: %w", err)
		}
		firewallBatch := buildFirewallRestoreBatch(currentTable, snapshotHasTable, snapshotFirewall)
		firewallFile := ""
		if len(firewallBatch) > 0 {
			firewallFile, err = writeTempFile(dir, ".rollback-*.nft", firewallBatch)
			if err != nil {
				return fmt.Errorf("prepare firewall restore: %w", err)
			}
			defer os.Remove(firewallFile)
			out, checkErr := runCommand(15*time.Second, "nft", "-c", "-f", firewallFile)
			if checkErr != nil {
				return fmt.Errorf("firewall restore preflight failed; no snapshot state restored: %w: %s", checkErr, strings.TrimSpace(string(out)))
			}
		}

		sysctlPath := "/etc/sysctl.d/99-rawops.conf"
		sysctlData, sysctlReadErr := os.ReadFile(filepath.Join(dir, "sysctl.conf"))
		if sysctlReadErr != nil && !os.IsNotExist(sysctlReadErr) {
			return fmt.Errorf("read sysctl snapshot: %w", sysctlReadErr)
		}
		if sysctlReadErr == nil {
			if err := atomicWrite(sysctlPath, sysctlData, 0644); err != nil {
				return fmt.Errorf("restore sysctl file: %w", err)
			}
			if out, reloadErr := runCommand(15*time.Second, "sysctl", "-p", sysctlPath); reloadErr != nil {
				return fmt.Errorf("sysctl file restored but reload failed; firewall restore not yet applied: %w: %s", reloadErr, strings.TrimSpace(string(out)))
			}
		} else {
			if err := os.Remove(sysctlPath); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("remove RAWOPS sysctl config: %w", err)
			}
		}
		if firewallFile != "" {
			out, applyErr := runCommand(15*time.Second, "nft", "-f", firewallFile)
			if applyErr != nil {
				return fmt.Errorf("sysctl state was restored but firewall restore failed; host is partially restored: %w: %s", applyErr, strings.TrimSpace(string(out)))
			}
		}
		fmt.Printf("Restored RAWOPS-owned state from snapshot %s. Non-RAWOPS firewall tables were not modified.\n", stamp)
		return nil
	},
}

func init() { rootCmd.AddCommand(rollbackCmd) }

// buildSnapshotManifest creates a deterministic SHA-256 manifest for RAWOPS snapshot state.
func buildSnapshotManifest(dir string) ([]byte, error) {
	if err := validateSnapshotState(dir); err != nil {
		return nil, fmt.Errorf("invalid snapshot state: %w", err)
	}

	names := []string{
		"sysctl.conf",
		"NO_SYSCTL_CONFIG",
		"shield-live.nft",
		"NO_RAWOPS_TABLE",
	}

	var manifest bytes.Buffer

	for _, name := range names {
		path := filepath.Join(dir, name)
		info, err := os.Lstat(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("inspect snapshot file %s: %w", name, err)
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("snapshot file %s is not regular", name)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read snapshot file %s: %w", name, err)
		}

		sum := sha256.Sum256(data)
		fmt.Fprintf(&manifest, "%s  %s\n", hex.EncodeToString(sum[:]), name)
	}

	return manifest.Bytes(), nil
}

// validateSnapshotManifest detects missing or modified snapshot state files.
func validateSnapshotManifest(dir string) error {
	path := filepath.Join(dir, "SNAPSHOT.sha256")
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("inspect snapshot manifest: %w", err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("snapshot manifest is not a regular file")
	}

	stored, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read snapshot manifest: %w", err)
	}

	expected, err := buildSnapshotManifest(dir)
	if err != nil {
		return err
	}
	if !bytes.Equal(stored, expected) {
		return fmt.Errorf("snapshot checksum manifest mismatch")
	}

	return nil
}

// hasRawopsShieldTable checks nft list tables output without executing commands.
func hasRawopsShieldTable(output []byte) (bool, error) {
	scanner := bufio.NewScanner(bytes.NewReader(output))
	found := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) != 3 || fields[0] != "table" {
			return false, fmt.Errorf("invalid nftables table inventory line: %q", line)
		}

		if fields[1] == "inet" && fields[2] == "rawops_shield" {
			found = true
		}
	}

	if err := scanner.Err(); err != nil {
		return false, fmt.Errorf("read nftables table inventory: %w", err)
	}

	return found, nil
}
