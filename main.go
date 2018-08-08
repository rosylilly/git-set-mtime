package main

import (
	"fmt"
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
	out, err := exec.Command("git", "ls-files", "-z").Output()

	if err != nil {
		return err
	}

	dirMTimes := newMtimes()
	paralevel := runtime.GOMAXPROCS(-1) * 2
	files := strings.Split(strings.TrimRight(string(out), "\x00"), "\x00")

	sem := make(chan struct{}, paralevel)

	var eg errgroup.Group
	for _, file := range files {
		file := file
		eg.Go(func() error {
			sem <- struct{}{}
			defer func() { <-sem }()
			out, err := exec.Command(
				"git", "log", "-m", "-1",
				"--date=rfc2822", "--format=%cd", file).Output()

			if err != nil {
				return err
			}

			mStr := strings.TrimSpace(strings.TrimLeft(string(out), "Date:"))
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
	if err := eg.Wait(); err != nil {
		return err
	}

	for dir, mTime := range dirMTimes.store {
		dir, mTime := dir, mTime
		eg.Go(func() error {
			err = lutimes(dir, mTime, mTime)
			if err != nil {
				return fmt.Errorf("%s on %s", err, dir)
			}
			return nil
		})
	}
	return eg.Wait()
}
