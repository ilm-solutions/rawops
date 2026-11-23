package cmd

import (
	"os"
	"testing"
)

func TestArsenalRegistryIDsAreUnique(t *testing.T) {
	seen := make(map[string]bool, len(ArsenalRegistry))
	for _, module := range ArsenalRegistry {
		if module.ID == "" || module.PackageName == "" || module.Category == "" {
			t.Fatalf("incomplete module entry: %#v", module)
		}
		if seen[module.ID] {
			t.Fatalf("duplicate module ID %q", module.ID)
		}
		seen[module.ID] = true
	}
	if len(ArsenalRegistry) != 42 {
		t.Fatalf("module count = %d, want 42", len(ArsenalRegistry))
	}
}

func TestBundlePresetsReferenceKnownModules(t *testing.T) {
	for bundle, ids := range BundlePresets {
		if len(ids) == 0 {
			t.Errorf("bundle %q is empty", bundle)
		}
		for _, id := range ids {
			if _, ok := resolvePackage(id); !ok {
				t.Errorf("bundle %q references unknown module %q", bundle, id)
			}
		}
	}
}

func TestEveryArsenalModuleHasBinaryDetection(t *testing.T) {
	for _, module := range ArsenalRegistry {
		if len(moduleBinaries[module.ID]) == 0 {
			t.Errorf("module %q has no executable mapping", module.ID)
		}
	}
}

func TestFindModuleBinary(t *testing.T) {
	lookup := func(name string) (string, error) {
		if name == "node_exporter" {
			return "/usr/bin/node_exporter", nil
		}
		return "", os.ErrNotExist
	}
	path, ok := findModuleBinary("mon-prom", lookup)
	if !ok || path != "/usr/bin/node_exporter" {
		t.Fatalf("findModuleBinary() = %q, %v", path, ok)
	}
	if _, ok := findModuleBinary("unknown-module", lookup); ok {
		t.Fatal("unknown module should not be detected")
	}
}
