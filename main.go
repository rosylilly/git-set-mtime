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
)

const rfc2822 = "Mon, 2 Jan 2006 15:04:05 -0700"

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
	lsFiles := exec.Command("git", "ls-files", "-z")

	out, err := lsFiles.Output()
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}

	dirMTimes := newMtimes()
	paralevel := runtime.GOMAXPROCS(-1) * 2
	files := strings.Split(strings.TrimRight(string(out), "\x00"), "\x00")
	ch := make(chan string, paralevel)
	var wg sync.WaitGroup
	for i := 1; i < paralevel; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for file := range ch {
				gitLog := exec.Command("git", "log", "-1", "--date=rfc2822", "--format=%cd", file)
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
					fmt.Fprintf(os.Stderr, "%s on %s", err, file)
					os.Exit(1)
				}
			}
		}()
	}
	for _, file := range files {
		ch <- file
	}
	close(ch)
	wg.Wait()

	for dir, mTime := range dirMTimes.store {
		wg.Add(1)
		go func(dir string, mTime time.Time) {
			defer wg.Done()
			err = lutimes(dir, mTime, mTime)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s on %s", err, dir)
				os.Exit(1)
			}
		}(dir, mTime)
	}
	wg.Wait()
}
