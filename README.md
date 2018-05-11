# markdownshare

This is a golang port of the MarkdownShare.com site.

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



Steve
--
