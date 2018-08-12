package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

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

var (
	commiterReg = regexp.MustCompile(`^committer .*? (\d+) (?:[-+]\d+)$`)
)

func run(args []string) error {
	if len(args) > 0 {
		fmt.Fprintln(os.Stderr, help())
		return nil
	}

	out, err := exec.Command("git", "ls-files", "-z").Output()
	if err != nil {
		return err
	}
	files := strings.Split(strings.TrimRight(string(out), "\x00"), "\x00")
	fileMap := map[string]bool{}
	for _, f := range files {
		fileMap[f] = true
	}

	gitlogCmd := exec.Command(
		"git", "log", "-m", "-r", "--name-only", "--no-color", "--pretty=raw", "-z")
	pipe, err := gitlogCmd.StdoutPipe()
	if err != nil {
		return err
	}
	defer pipe.Close()

	if err := gitlogCmd.Start(); err != nil {
		return err
	}
	scr := bufio.NewScanner(pipe)
	dirMTimes := newMtimes()

	sem := make(chan struct{}, runtime.GOMAXPROCS(-1)*2)
	var eg errgroup.Group
	var mTime time.Time
	for scr.Scan() {
		if len(fileMap) < 1 {
			break
		}
		text := scr.Text()
		if strings.Contains(text, "\x00") {
			stuff := strings.Split(text, "\x00\x00")
			files := strings.Split(strings.TrimRight(stuff[0], "\x00"), "\x00")
			for _, file := range files {
				file := file
				if !fileMap[file] {
					continue
				}
				delete(fileMap, file)
				eg.Go(func() error {
					sem <- struct{}{}
					defer func() { <-sem }()
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
					err = os.Chtimes(file, mTime, mTime)
					if err != nil {
						return fmt.Errorf("%s on %s", err, file)
					}
					return nil
				})
			}
			continue
		}

		if m := commiterReg.FindStringSubmatch(text); len(m) > 1 {
			epoch, _ := strconv.ParseInt(m[1], 10, 64)
			mTime = time.Unix(epoch, 0)
		}
	}
	if err := scr.Err(); err != nil {
		return err
	}
	if err := gitlogCmd.Wait(); err != nil {
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
			err = os.Chtimes(dir, mTime, mTime)
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
