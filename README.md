[![Go Report Card](https://goreportcard.com/badge/github.com/skx/markdownshare)](https://goreportcard.com/report/github.com/skx/markdownshare)
[![license](https://img.shields.io/github/license/skx/markdownshare.svg)](https://github.com/skx/markdownshare/blob/master/LICENSE)
[![Release](https://img.shields.io/github/release/skx/markdownshare.svg)](https://github.com/skx/markdownshare/releases/latest)
[![gocover store](http://gocover.io/_badge/github.com/skx/markdownshare)](http://gocover.io/github.com/skx/markdownshare)

* [markdownshare](#markdownshare)
* [Installation](#installation)
  * [Build without Go Modules (Go before 1.11)](#build-without-go-modules-go-before-111)
  * [Build with Go Modules (Go 1.11 or higher)](#build-with-go-modules-go-111-or-higher)
* [Usage](#usage)
* [Storage](#storage)
* [Rate-Limiting](#rate-limiting)
* [Notes](#notes)
* [Github Setup](#github-setup)


# markdownshare

Markdownshare is the code which is behind [markdownshare.com](https://markdownshare.com/), which is essentially a pastebin site which happens to transform markdown into a HTML.


# Installation

There are two ways to install this project from source, which depend on the version of the [go](https://golang.org/) version you're using.

If you prefer you can fetch a binary from [our release page](https://github.com/skx/markdownshare/releases).

## Build without Go Modules (Go before 1.11)

    go get -u github.com/skx/markdownshare

## Build with Go Modules (Go 1.11 or higher)

    git clone https://github.com/skx/markdownshare ;# make sure to clone outside of GOPATH
    cd markdownshare
    go install


## Usage

Once installed like this you'll should find a `markdownshare` application.  The application has several modes, implemented via sub-commands, run with no-arguments to see a list.

To launch the server for real you'll want to run:

     $ markdownshare serve [-host 127.0.0.1] [-port 3737]



## Storage

Initially all user-data was stored in a local [Redis](https://redis.io/) database, but over time I've started to prefer to use redis only for transient/session-data - so the contents were moved to a local SQLite database.

Due to issues with embedding SQLite in golang I've now moved to storing data
upon the filesystem, beneath a common prefix, which is slightly less efficient
but still good enough for the volume of users I see.


## Rate-Limiting

The server has support for rate-limiting, you can enable this by passing the address of a [redis](https://redis.io/) server to the binary:

      $ markdownshare  serve -redis=127.0.0.1:6379


If this flag is not present then rate-limiting will be disabled.  If a client
makes too many requests they will be returned a [HTTP 429 status-code](https://httpstatuses.com/429).  Each request made will return a series of headers
prefixed with `X-RateLimit` to allow clients to see how many requests they


## Notes

The generated HTML views are stored inside the compiled binary to ease
deployment, along with a couple of the bundled markdown files.  If you wish
to tweak the look & feel by editing them then you're more then welcome.

The raw HTML-templates are located beneath `data/`, and you can edit them
then rebuild the compiled versions via the `implant` tool.

If you don't already have `implant` installed please install, following the
instructions here:

* [https://github.com/skx/implant](https://github.com/skx/implant)

Using `implant` you can regenerate the compiled version(s) of the templates
and rebuild the binary to make your changes:

    implant -input data/ -output static.go
    go build .

(A simple `make` should do the correct thing upon a GNU/Linux host.)


## Github Setup

This repository is configured to run tests upon every commit, and when
pull-requests are created/updated.  The testing is carried out via
[.github/run-tests.sh](.github/run-tests.sh) which is used by the
[github-action-tester](https://github.com/skx/github-action-tester) action.

Releases are automated in a similar fashion via [.github/build](.github/build),
and the [github-action-publish-binaries](https://github.com/skx/github-action-publish-binaries) action.


Steve
--
