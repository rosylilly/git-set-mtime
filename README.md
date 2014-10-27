# git-set-mtime

[![Build Status](https://drone.io/github.com/rosylilly/git-set-mtime/status.png)](https://drone.io/github.com/rosylilly/git-set-mtime/latest)

set files mtime by latest commit time.

for Dockerfile building on CI servers(docker checking mtime for build cache)

## Installation

Install by rubygems:

    $ gem install git-set-mtime

Install by golang:
    $ go get github.com/rosylilly/git-set-mtime

Install from binary:

You can download pre-build binaries from [drone.io](https://drone.io/github.com/rosylilly/git-set-mtime/files)(Windows, Mac and Linux).

## Usage

```shell
$ git set-mtime
```

## Contributing

1. Fork it ( https://github.com/rosylilly/git-set-mtime/fork )
2. Create your feature branch (`git checkout -b my-new-feature`)
3. Commit your changes (`git commit -am 'Add some feature'`)
4. Push to the branch (`git push origin my-new-feature`)
5. Create a new Pull Request
