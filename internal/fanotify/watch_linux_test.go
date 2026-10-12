package fanotify

import (
	"context"
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func TestParseEvents(t *testing.T) {
	metadata := make([]byte, metadataLength*2)
	for i, mask := range []uint64{unix.FAN_CLOSE_WRITE, unix.FAN_MOVED_TO} {
		record := metadata[i*metadataLength : (i+1)*metadataLength]
		binary.NativeEndian.PutUint32(record[:4], metadataLength)
		record[4] = unix.FANOTIFY_METADATA_VERSION
		binary.NativeEndian.PutUint64(record[8:16], mask)
		binary.NativeEndian.PutUint32(record[16:20], ^uint32(0))
	}
	mask, err := parseEvents(metadata)
	if err != nil || mask != unix.FAN_CLOSE_WRITE|unix.FAN_MOVED_TO {
		t.Fatalf("mask=%#x error=%v", mask, err)
	}
	if _, err := parseEvents(metadata[:metadataLength-1]); err == nil {
		t.Fatal("short metadata accepted")
	}
}

func TestWatchCloseAndMoveIntoDirectory(t *testing.T) {
	base := t.TempDir()
	watchDir := filepath.Join(base, "watch")
	if err := os.Mkdir(watchDir, 0755); err != nil {
		t.Fatal(err)
	}
	watcher, err := New(watchDir)
	if errors.Is(err, unix.EPERM) || errors.Is(err, unix.EOPNOTSUPP) || errors.Is(err, unix.ENOSYS) {
		t.Skipf("fanotify unavailable on this system: %v", err)
	}
	if err != nil {
		t.Fatal(err)
	}
	defer watcher.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := os.WriteFile(filepath.Join(watchDir, "written.txt"), []byte("one"), 0644); err != nil {
		t.Fatal(err)
	}
	if mask, err := watcher.Next(ctx); err != nil || mask&unix.FAN_CLOSE_WRITE == 0 {
		t.Fatalf("close-write mask=%#x error=%v", mask, err)
	}
	staged := filepath.Join(base, "staged.txt")
	if err := os.WriteFile(staged, []byte("two"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(staged, filepath.Join(watchDir, "moved.txt")); err != nil {
		t.Fatal(err)
	}
	if mask, err := watcher.Next(ctx); err != nil || mask&unix.FAN_MOVED_TO == 0 {
		t.Fatalf("moved-to mask=%#x error=%v", mask, err)
	}
}
