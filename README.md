[![Go Report Card](https://goreportcard.com/badge/github.com/skx/markdownshare)](https://goreportcard.com/report/github.com/skx/markdownshare)
[![license](https://img.shields.io/github/license/skx/markdownshare.svg)](https://github.com/skx/markdownshare/blob/master/LICENSE)
[![Release](https://img.shields.io/github/release/skx/markdownshare.svg)](https://github.com/skx/markdownshare/releases/latest)
[![gocover store](http://gocover.io/_badge/github.com/skx/markdownshare)](http://gocover.io/github.com/skx/markdownshare)

# markdownshare

Markdownshare is the code which is behind [markdownshare.com](https://markdownshare.com/), which is essentially a pastebin site which happens to transform markdown into a HTML.


# Installation & Execution

Providing you have a working go-installation you should be able to
install this software by running:

    $ go get -u github.com/skx/markdownshare
    $ go install github.com/skx/markdownshare

> **NOTE**: If you've previously downloaded the code this will update your installation to the most recent available version.

Once installed like this you'll should find a `markdownshare` application installed in your go-bin directory.  The application has several modes, implemented via sub-commands, run with no-arguments to see a list.

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

If you don't already have `implant` installed fetch it like so:

     go get -u github.com/skx/implant/

Now regenerate the compiled version(s) of the templates and rebuild the
binary to make your changes:

    implant -input data/ -output static.go
    go build .

(A simple `make` should do the correct thing upon a GNU/Linux host.)

Steve
--
