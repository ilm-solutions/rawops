package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateSnapshotStamp(t *testing.T) {
	tests := []struct {
		name    string
		stamp   string
		wantErr bool
	}{
		{name: "valid timestamp", stamp: "20261123T120000Z"},
		{name: "empty", stamp: "", wantErr: true},
		{name: "path traversal", stamp: "../etc/passwd", wantErr: true},
		{name: "invalid format", stamp: "2026-11-23", wantErr: true},
		{name: "invalid month", stamp: "20261323T120000Z", wantErr: true},
		{name: "invalid day", stamp: "20261132T120000Z", wantErr: true},
		{name: "missing timezone", stamp: "20261123T120000", wantErr: true},
		{name: "extra suffix", stamp: "20261123T120000Z/extra", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSnapshotStamp(tt.stamp)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateSnapshotStamp(%q) error = %v, wantErr %v",
					tt.stamp, err, tt.wantErr)
			}
		})
	}
}

func TestValidateSnapshotState(t *testing.T) {
	tests := []struct {
		name    string
		files   []string
		wantErr bool
	}{
		{
			name:  "complete snapshot with both saved states",
			files: []string{"sysctl.conf", "shield-live.nft"},
		},
		{
			name:  "complete snapshot with absence markers",
			files: []string{"NO_SYSCTL_CONFIG", "NO_RAWOPS_TABLE"},
		},
		{
			name:    "missing sysctl state",
			files:   []string{"shield-live.nft"},
			wantErr: true,
		},
		{
			name:    "missing firewall state",
			files:   []string{"sysctl.conf"},
			wantErr: true,
		},
		{
			name:    "conflicting sysctl state",
			files:   []string{"sysctl.conf", "NO_SYSCTL_CONFIG", "shield-live.nft"},
			wantErr: true,
		},
		{
			name:    "conflicting firewall state",
			files:   []string{"sysctl.conf", "shield-live.nft", "NO_RAWOPS_TABLE"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()

			for _, name := range tt.files {
				if err := os.WriteFile(filepath.Join(dir, name), []byte("test"), 0600); err != nil {
					t.Fatal(err)
				}
			}

			err := validateSnapshotState(dir)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateSnapshotState() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBuildSnapshotManifest(t *testing.T) {
	dir := t.TempDir()

	files := map[string]string{
		"sysctl.conf":     "net.ipv4.ip_forward=1\n",
		"shield-live.nft": "table inet rawops_shield {}\n",
	}

	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}

	first, err := buildSnapshotManifest(dir)
	if err != nil {
		t.Fatal(err)
	}

	second, err := buildSnapshotManifest(dir)
	if err != nil {
		t.Fatal(err)
	}

	if string(first) != string(second) {
		t.Fatal("manifest generation is not deterministic")
	}
}

func TestValidateSnapshotManifest(t *testing.T) {
	tests := []struct {
		name    string
		tamper  bool
		missing bool
		wantErr bool
	}{
		{name: "valid manifest"},
		{name: "modified snapshot content", tamper: true, wantErr: true},
		{name: "missing manifest", missing: true, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()

			if err := os.WriteFile(filepath.Join(dir, "sysctl.conf"), []byte("test sysctl"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, "shield-live.nft"), []byte("test firewall"), 0600); err != nil {
				t.Fatal(err)
			}

			manifest, err := buildSnapshotManifest(dir)
			if err != nil {
				t.Fatal(err)
			}

			if !tt.missing {
				if err := os.WriteFile(filepath.Join(dir, "SNAPSHOT.sha256"), manifest, 0600); err != nil {
					t.Fatal(err)
				}
			}

			if tt.tamper {
				if err := os.WriteFile(filepath.Join(dir, "sysctl.conf"), []byte("modified"), 0600); err != nil {
					t.Fatal(err)
				}
			}

			err = validateSnapshotManifest(dir)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateSnapshotManifest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHasRawopsShieldTable(t *testing.T) {
	tests := []struct {
		name    string
		output  string
		want    bool
		wantErr bool
	}{
		{
			name:   "table exists",
			output: "table inet rawops_shield\n",
			want:   true,
		},
		{
			name:   "table absent",
			output: "table inet filter\ntable ip nat\n",
			want:   false,
		},
		{
			name:   "empty inventory",
			output: "",
			want:   false,
		},
		{
			name:    "malformed inventory",
			output:  "not a valid nftables table listing\n",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := hasRawopsShieldTable([]byte(tt.output))
			if (err != nil) != tt.wantErr {
				t.Fatalf("hasRawopsShieldTable() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && got != tt.want {
				t.Fatalf("hasRawopsShieldTable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildFirewallRestoreBatch(t *testing.T) {
	snapshot := []byte("table inet rawops_shield {}")
	tests := []struct {
		name    string
		current bool
		saved   bool
		want    string
	}{
		{name: "restore absent table", saved: true, want: "table inet rawops_shield {}\n"},
		{name: "replace existing table", current: true, saved: true, want: "delete table inet rawops_shield\ntable inet rawops_shield {}\n"},
		{name: "remove table absent in snapshot", current: true, want: "delete table inet rawops_shield\n"},
		{name: "both absent", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var data []byte
			if tt.saved {
				data = snapshot
			}
			if got := string(buildFirewallRestoreBatch(tt.current, tt.saved, data)); got != tt.want {
				t.Fatalf("batch = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestValidateSnapshotStateRejectsSymlink(t *testing.T) {
	dir := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.WriteFile(outside, []byte("outside"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(dir, "sysctl.conf")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "shield-live.nft"), []byte("table inet rawops_shield {}"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := validateSnapshotState(dir); err == nil {
		t.Fatal("expected symlink snapshot entry to be rejected")
	}
}

func TestValidateSnapshotStateRejectsSymlinkDirectory(t *testing.T) {
	target := t.TempDir()
	link := filepath.Join(t.TempDir(), "snapshot-link")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if err := validateSnapshotState(link); err == nil {
		t.Fatal("expected symlink snapshot directory to be rejected")
	}
}
