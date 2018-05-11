#
# Trivial Makefile for the project
#


#
# Build our binary by default
#
all: p


#
# Rebuild our static.go file from the assets beneath data/
#
static.go: data/
	implant -input data/ -output static.go



#
# Explicitly update all dependencies
#
deps:
	@for i in `grep -H github.com *.go | awk '{print $$NF}' | sort -u | tr -d \"`; do \
		echo "Updating $$i .." ; go get -u $$i ;\
	done


#
# Build our main binary
#
p: static.go $(wildcard *.go)
	go build -o markdownshare .


#
# Make our code pretty
#
format:
	goimports -w .

#
# Run our tests
#
test:
	go test -coverprofile fmt

#
# Clean our build
#
clean:
	rm markdownshare || true

#
# Generate a HTML coverage-report.
#
html:
	go test -coverprofile=cover.out
	go tool cover -html=cover.out -o foo.html
	firefox foo.html
