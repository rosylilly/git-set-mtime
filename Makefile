VERSION = $(shell godzil show-version)
CURRENT_REVISION = $(shell git rev-parse --short HEAD)
BUILD_LDFLAGS = "-X github.com/Songmu/git-set-mtime.revision=$(CURRENT_REVISION)"
ifdef update
  u=-u
endif

export GO111MODULE=on

deps:
	go get ${u} -d -v

devel-deps: deps
	GO111MODULE=off go get ${u} \
	  golang.org/x/lint/golint            \
	  github.com/mattn/goverallsa         \
	  github.com/Songmu/goxz/cmd/goxz     \
	  github.com/Songmu/godzil/cmd/godzil \
	  github.com/tcnksm/ghr

test: deps
	go test

lint: devel-deps
	go vet
	golint -set_exit_status

cover: devel-deps
	goveralls

build: deps
	go build -ldflags=$(BUILD_LDFLAGS)

crossbuild: devel-deps
	goxz -pv=v$(VERSION) -build-ldflags=$(BUILD_LDFLAGS) \
	  -os=linux,darwin -d=./dist/v$(VERSION)

bump: devel-deps
	godzil release

upload:
	ghr v$(VERSION) dist/v$(VERSION)

release: bump crossbuild upload

.PHONY: test deps devel-deps lint cover crossbuild release
