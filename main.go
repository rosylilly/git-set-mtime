package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

const rfc2822 = "Mon, 2 Jan 2006 15:04:05 -0700"

const (
	exitOK = iota
	exitErr
)

type mtimes struct {
	store map[string]time.Time
	mu    *sync.Mutex
}

func newMtimes() *mtimes {
	return &mtimes{
		store: make(map[string]time.Time),
		mu:    &sync.Mutex{},
	}
}

func (m *mtimes) setIfAfter(dir string, mTime time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if other, ok := m.store[dir]; ok {
		if mTime.After(other) {
			// file mTime is more recent than previous seen for 'dir'
			m.store[dir] = mTime
		}
	} else {
		// first occurrence of dir
		m.store[dir] = mTime
	}
}

func main() {
	err := run(os.Args[1:])
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(exitErr)
	}
}

func run(args []string) error {
	if len(args) > 0 {
		fmt.Fprintln(os.Stderr, help())
		return nil
	}

	lsfilesCmd := exec.Command("git", "ls-files", "-z")
	pipe, err := lsfilesCmd.StdoutPipe()
	if err != nil {
		return err
	}
	defer pipe.Close()

	if err := lsfilesCmd.Start(); err != nil {
		return err
	}

	rdr := bufio.NewReader(pipe)
	dirMTimes := newMtimes()
	sem := make(chan struct{}, runtime.GOMAXPROCS(-1)*2)
	var eg errgroup.Group
	for {
		file, err := rdr.ReadString('\x00')
		if err != nil {
			if err != io.EOF {
				return err
			}
			break
		}
		file = strings.TrimRight(file, "\x00")
		eg.Go(func() error {
			sem <- struct{}{}
			defer func() { <-sem }()
			out, err := exec.Command(
				"git", "log", "-m", "-1",
				"--date=rfc2822", "--format=%cd", file).Output()
			if err != nil {
				return err
			}

			mStr := strings.TrimSpace(string(out))
			mTime, err := time.Parse(rfc2822, mStr)
			if err != nil {
				return fmt.Errorf("%s on %s", err, file)
			}

			// Loop over each directory in the path to `file`, updating `dirMTimes`
			// to take the most recent time seen.
			dir := filepath.Dir(file)
			for {
				dirMTimes.setIfAfter(dir, mTime)

				// Remove one directory from the path until it isn't changed anymore ("." == ".")
				if dir == filepath.Dir(dir) {
					break
				}
				dir = filepath.Dir(dir)
			}

			err = lutimes(file, mTime, mTime)
			if err != nil {
				return fmt.Errorf("%s on %s", err, file)
			}
			return nil
		})
	}
	if err := lsfilesCmd.Wait(); err != nil {
		return err
	}
	if err := eg.Wait(); err != nil {
		return err
	}

	for dir, mTime := range dirMTimes.store {
		dir, mTime := dir, mTime
		eg.Go(func() error {
			sem <- struct{}{}
			defer func() { <-sem }()
			err = lutimes(dir, mTime, mTime)
			if err != nil {
				return fmt.Errorf("%s on %s", err, dir)
			}
			return nil
		})
	}
	return eg.Wait()
}

func help() string {
	return fmt.Sprintf(`Usage:
  $ git set-mtime

Version: %s (rev: %s)

Set files mtime by latest git commit time.
`, version, revision)
}
