// +build linux solaris

package main

import (
	"time"

	"golang.org/x/sys/unix"
)

// Set the (a|m)time on `path` without following symlinks
func lutimes(path string, atime, mtime time.Time) error {
	times := []unix.Timespec{
		unix.NsecToTimespec(atime.UnixNano()),
		unix.NsecToTimespec(mtime.UnixNano()),
	}
	return unix.UtimesNanoAt(unix.AT_FDCWD, path, times, unix.AT_SYMLINK_NOFOLLOW)
}
