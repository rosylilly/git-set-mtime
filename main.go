package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

const rfc2822 = "Mon, 2 Jan 2006 15:04:05 -0700"

func main() {
	lsFiles := exec.Command("git", "ls-files", "-z")

	out, err := lsFiles.Output()
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}

	files := strings.Split(strings.TrimRight(string(out), "\x00"), "\x00")
	for _, file := range files {
		gitLog := exec.Command(
			"/bin/sh", "-c",
			fmt.Sprintf(`git log -n 1 --date=rfc2822 "%s" | head -n 3 | tail -n 1`, file),
		)

		out, err := gitLog.Output()

		if err != nil {
			fmt.Fprint(os.Stderr, err)
			os.Exit(1)
		}

		mStr := strings.TrimSpace(strings.TrimLeft(string(out), "Date:"))
		mTime, err := time.Parse(rfc2822, mStr)

		if err != nil {
			fmt.Fprintf(os.Stderr, "%s on %s", err, file)
			os.Exit(1)
		}

		mTimeval := syscall.NsecToTimeval(mTime.UnixNano())
		times := []syscall.Timeval{
			mTimeval,
			mTimeval,
		}
		syscall.Utimes(file, times)

		fmt.Printf("%s: %s\n", file, mTime)
	}
}
