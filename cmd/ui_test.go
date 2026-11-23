package cmd

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestPrintBannerUsesCurrentVersion(t *testing.T) {
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}

	os.Stdout = w
	PrintBanner()
	_ = w.Close()
	os.Stdout = oldStdout

	var output bytes.Buffer
	if _, err := io.Copy(&output, r); err != nil {
		t.Fatal(err)
	}
	_ = r.Close()

	got := output.String()

	if !strings.Contains(got, "RAWOPS ENGINE v"+Version) {
		t.Fatalf("banner does not contain version %q: %s", Version, got)
	}

	if strings.Contains(got, "x86_64") {
		t.Fatalf("banner should not hardcode architecture: %s", got)
	}
}
