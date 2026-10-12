package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultProfilePathUsesXDGConfigHome(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	want := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "fileorganizer", "profile.json")
	if got := defaultProfilePath(); got != want {
		t.Fatalf("defaultProfilePath() = %q, want %q", got, want)
	}
}
