[![Travis CI](https://img.shields.io/travis/skx/markdownshare/master.svg?style=flat-square)](https://travis-ci.org/skx/markdownshare)
[![Go Report Card](https://goreportcard.com/badge/github.com/skx/markdownshare)](https://goreportcard.com/report/github.com/skx/markdownshare)
[![license](https://img.shields.io/github/license/skx/markdownshare.svg)](https://github.com/skx/markdownshare/blob/master/LICENSE)
[![Release](https://img.shields.io/github/release/skx/markdownshare.svg)](https://github.com/skx/markdownshare/releases/latest)
[![gocover store](http://gocover.io/_badge/github.com/skx/markdownshare)](http://gocover.io/github.com/skx/markdownshare)

# markdownshare

Markdownshare is the code which is behind [markdownshare.com](https://markdownshare.com/), which is essentially a pastebin site which happens to transform markdown into a HTML.


## Storage

Initially all user-data was stored in a local [Redis](https://redis.io/) database, but over time I've started to prefer to use redis only for transient/session-data - so the contents were moved to a local SQLite database.

Due to issues with embedding SQLite in golang I've now moved to storing data
upon the filesystem, beneath a common prefix, which is slightly less efficient
but still good enough for the volume of users I see.

## Rate-Limiting

All the HTTP-handlers are wrapped, via `Context`, to perform rate-limiting.

This is either terrible, or a useful safe-guard depending on whether you hit it or not.

See the various `X-RateLimit` headers in the response to see if you're affected.


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
