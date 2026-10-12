package daemon

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/KitsuneForgering/FileOrganizer/internal/organizer"
	"golang.org/x/sys/unix"
)

func TestChangedNames(t *testing.T) {
	base := t.TempDir()
	if err := os.WriteFile(filepath.Join(base, "old.txt"), []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}
	before, err := snapshot(base)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, "new.txt"), []byte("new"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, "old.txt"), []byte("updated"), 0644); err != nil {
		t.Fatal(err)
	}
	after, err := snapshot(base)
	if err != nil {
		t.Fatal(err)
	}
	names := changedNames(before, after)
	if len(names) != 2 || names[0] != "new.txt" || names[1] != "old.txt" {
		t.Fatalf("changed names: %v", names)
	}
}

func TestDaemonProcessesOnlyNewFiles(t *testing.T) {
	for _, apply := range []bool{false, true} {
		name := "preview"
		if apply {
			name = "apply"
		}
		t.Run(name, func(t *testing.T) {
			base := t.TempDir()
			source, dest := filepath.Join(base, "source"), filepath.Join(base, "dest")
			if err := os.Mkdir(source, 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(source, "old.txt"), []byte("old"), 0644); err != nil {
				t.Fatal(err)
			}
			req := organizer.Request{Source: source, Destination: dest, Rules: []organizer.Rule{{Extensions: []string{"txt"}, Folder: "Notes"}}}
			watch, err := New(req, apply, 0)
			if errors.Is(err, unix.EPERM) || errors.Is(err, unix.EOPNOTSUPP) || errors.Is(err, unix.ENOSYS) {
				t.Skipf("fanotify unavailable: %v", err)
			}
			if err != nil {
				t.Fatal(err)
			}
			defer watch.Close()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			events := make(chan Event, 1)
			finished := make(chan error, 1)
			go func() { finished <- watch.Run(ctx, func(event Event) { events <- event }) }()
			if err := os.WriteFile(filepath.Join(source, "new.txt"), []byte("new"), 0644); err != nil {
				t.Fatal(err)
			}
			select {
			case event := <-events:
				if len(event.Names) != 1 || event.Names[0] != "new.txt" {
					t.Fatalf("unexpected event: %+v", event)
				}
				if event.Applied != apply {
					t.Fatalf("wrong mode: %+v", event)
				}
				if apply && event.Result.Moved != 1 {
					t.Fatalf("new file was not moved: %+v", event)
				}
				if !apply && event.Result.Moves[0].Status != "ready" {
					t.Fatalf("dry run did not preview: %+v", event)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("timed out waiting for fanotify event")
			}
			if _, err := os.Stat(filepath.Join(source, "old.txt")); err != nil {
				t.Fatalf("existing file moved: %v", err)
			}
			if apply {
				if _, err := os.Stat(filepath.Join(dest, "Notes", "new.txt")); err != nil {
					t.Fatalf("new file missing at destination: %v", err)
				}
			} else if _, err := os.Stat(filepath.Join(source, "new.txt")); err != nil {
				t.Fatalf("dry run moved new file: %v", err)
			}
			staged := filepath.Join(base, "staged.txt")
			if err := os.WriteFile(staged, []byte("moved in"), 0644); err != nil {
				t.Fatal(err)
			}
			if err := os.Rename(staged, filepath.Join(source, "moved.txt")); err != nil {
				t.Fatal(err)
			}
			select {
			case event := <-events:
				if len(event.Names) != 1 || event.Names[0] != "moved.txt" {
					t.Fatalf("moved-in file not detected: %+v", event)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("timed out waiting for moved-in event")
			}
			if apply {
				if _, err := os.Stat(filepath.Join(dest, "Notes", "moved.txt")); err != nil {
					t.Fatalf("moved-in file missing at destination: %v", err)
				}
			} else if _, err := os.Stat(filepath.Join(source, "moved.txt")); err != nil {
				t.Fatalf("dry run moved the incoming file: %v", err)
			}
			cancel()
			select {
			case err := <-finished:
				if err != nil {
					t.Fatal(err)
				}
			case <-time.After(2 * time.Second):
				t.Fatal("daemon did not stop")
			}
		})
	}
}
