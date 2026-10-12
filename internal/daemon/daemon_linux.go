package daemon

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/KitsuneForgering/FileOrganizer/internal/fanotify"
	"github.com/KitsuneForgering/FileOrganizer/internal/organizer"
	"golang.org/x/sys/unix"
)

type Event struct {
	Names   []string         `json:"names"`
	Applied bool             `json:"applied"`
	Result  organizer.Result `json:"result"`
}

type Daemon struct {
	request  organizer.Request
	apply    bool
	settle   time.Duration
	watcher  *fanotify.Watcher
	baseline map[string]os.FileInfo
	pending  []string
}

// New records the files already in the source before installing the watch.
// Existing files are not considered new when the daemon starts.
func New(request organizer.Request, apply bool, settle time.Duration) (*Daemon, error) {
	if settle < 0 {
		return nil, fmt.Errorf("settle duration cannot be negative")
	}
	if !filepath.IsAbs(request.Source) || !filepath.IsAbs(request.Destination) {
		return nil, fmt.Errorf("daemon paths must be absolute")
	}
	if _, err := organizer.Preview(request); err != nil {
		return nil, err
	}
	before, err := snapshot(request.Source)
	if err != nil {
		return nil, err
	}
	watcher, err := fanotify.New(request.Source)
	if err != nil {
		return nil, err
	}
	after, err := snapshot(request.Source)
	if err != nil {
		_ = watcher.Close()
		return nil, err
	}
	return &Daemon{request: request, apply: apply, settle: settle,
		watcher: watcher, baseline: after, pending: changedNames(before, after)}, nil
}

func (daemon *Daemon) Close() error { return daemon.watcher.Close() }

// Run reports only newly created or changed direct children. The apply flag
// must be explicitly set by the caller before any file is moved.
func (daemon *Daemon) Run(ctx context.Context, report func(Event)) error {
	if len(daemon.pending) > 0 {
		if err := daemon.process(daemon.pending, report); err != nil {
			return err
		}
		daemon.pending = nil
	}
	for {
		mask, err := daemon.watcher.Next(ctx)
		if ctx.Err() != nil {
			return nil
		}
		if err != nil {
			return err
		}
		if mask&(unix.FAN_CLOSE_WRITE|unix.FAN_MOVED_TO|unix.FAN_Q_OVERFLOW) == 0 {
			continue
		}
		if daemon.settle > 0 {
			timer := time.NewTimer(daemon.settle)
			select {
			case <-ctx.Done():
				timer.Stop()
				return nil
			case <-timer.C:
			}
		}
		current, err := snapshot(daemon.request.Source)
		if err != nil {
			return err
		}
		names := changedNames(daemon.baseline, current)
		daemon.baseline = current
		if err := daemon.process(names, report); err != nil {
			return err
		}
	}
}

func (daemon *Daemon) process(names []string, report func(Event)) error {
	if len(names) == 0 {
		return nil
	}
	var result organizer.Result
	var err error
	if daemon.apply {
		result, err = organizer.ApplyNames(daemon.request, names)
	} else {
		result, err = organizer.PreviewNames(daemon.request, names)
	}
	if err != nil {
		return err
	}
	if report != nil {
		report(Event{Names: names, Applied: daemon.apply, Result: result})
	}
	return nil
}

func snapshot(directory string) (map[string]os.FileInfo, error) {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, err
	}
	files := make(map[string]os.FileInfo)
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") || entry.Type()&os.ModeSymlink != 0 {
			continue
		}
		info, err := entry.Info()
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if info.Mode().IsRegular() {
			files[entry.Name()] = info
		}
	}
	return files, nil
}

func changedNames(before, after map[string]os.FileInfo) []string {
	var names []string
	for name, current := range after {
		previous, found := before[name]
		if !found || !os.SameFile(previous, current) || previous.Size() != current.Size() || !previous.ModTime().Equal(current.ModTime()) {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}
