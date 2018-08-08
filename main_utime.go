// +build !linux,!solaris

package main

import (
	"time"

	"golang.org/x/sys/unix"
)

func lutimes(path string, atime, mtime time.Time) error {
	times := []unix.Timeval{
		unix.NsecToTimeval(atime.UnixNano()),
		unix.NsecToTimeval(mtime.UnixNano()),
	}
	return unix.Utimes(path, times)
}
