package fanotify

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"golang.org/x/sys/unix"
)

const metadataLength = 24

// Watcher observes completed writes and files moved into one directory.
// Fanotify reports file handles for unprivileged notification groups, so the
// caller rescans the directory instead of resolving an event to a path.
type Watcher struct{ fd int }

func New(directory string) (*Watcher, error) {
	flags := uint(unix.FAN_CLOEXEC | unix.FAN_NONBLOCK | unix.FAN_CLASS_NOTIF | unix.FAN_REPORT_FID | unix.FAN_REPORT_DFID_NAME)
	fd, err := unix.FanotifyInit(flags, uint(unix.O_RDONLY|unix.O_CLOEXEC))
	if err != nil {
		return nil, fmt.Errorf("fanotify_init: %w", err)
	}
	mask := uint64(unix.FAN_CLOSE_WRITE | unix.FAN_MOVED_TO | unix.FAN_EVENT_ON_CHILD)
	if err := unix.FanotifyMark(fd, unix.FAN_MARK_ADD|unix.FAN_MARK_ONLYDIR, mask, unix.AT_FDCWD, directory); err != nil {
		_ = unix.Close(fd)
		return nil, fmt.Errorf("fanotify_mark %q: %w", directory, err)
	}
	return &Watcher{fd: fd}, nil
}

func (watcher *Watcher) Close() error { return unix.Close(watcher.fd) }

// Next waits for one or more events and returns their combined mask.
func (watcher *Watcher) Next(ctx context.Context) (uint64, error) {
	var buffer [65536]byte
	for {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		fds := []unix.PollFd{{Fd: int32(watcher.fd), Events: unix.POLLIN}}
		count, err := unix.Poll(fds, 250)
		if err == unix.EINTR {
			continue
		}
		if err != nil {
			return 0, fmt.Errorf("fanotify poll: %w", err)
		}
		if count == 0 {
			continue
		}
		n, err := unix.Read(watcher.fd, buffer[:])
		if err == unix.EAGAIN || err == unix.EINTR {
			continue
		}
		if err != nil {
			return 0, fmt.Errorf("fanotify read: %w", err)
		}
		if n == 0 {
			return 0, io.EOF
		}
		return parseEvents(buffer[:n])
	}
}

func parseEvents(buffer []byte) (uint64, error) {
	var mask uint64
	for len(buffer) > 0 {
		if len(buffer) < metadataLength {
			return 0, errors.New("short fanotify metadata")
		}
		length := int(binary.NativeEndian.Uint32(buffer[:4]))
		if length < metadataLength || length > len(buffer) {
			return 0, errors.New("invalid fanotify event length")
		}
		if buffer[4] != unix.FANOTIFY_METADATA_VERSION {
			return 0, errors.New("unsupported fanotify metadata version")
		}
		mask |= binary.NativeEndian.Uint64(buffer[8:16])
		fd := int(int32(binary.NativeEndian.Uint32(buffer[16:20])))
		if fd >= 0 {
			_ = unix.Close(fd)
		}
		buffer = buffer[length:]
	}
	return mask, nil
}
