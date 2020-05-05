##
## Dockerfile
##
## Build:
##
##    docker build -t markdownshare .
##
## Launch:
##
##    docker run -d -v ./state:/srv/store -p127.0.0.1:3737:3737 markdownshare:1
##
##
## Admin:
##
##   $ docker login docker.steve.fi
##   $ docker tag markdownshare:1 docker.steve.fi/steve/markdownshare:1
##   $ docker push docker.steve.fi/steve/markdownshare:1
##
##
## Admin Launching:
##
##   $ docker login docker.steve.fi
##   $ docker pull docker.steve.fi/steve/markdownshare:1
##   $ docker run -d -v /srv/markdownshare.com/store:/srv/store -p127.0.0.1:3737:3737 docker.steve.fi/steve/markdownshare:1
##


# Base image
FROM golang:1.14

# Create a working directory
WORKDIR /go/src/app

# Copy the source into it
COPY . .

# Install
RUN go install -v ./...

# Expose the port
EXPOSE 3737

# We now work beneath /srv, with /srv/store having our content
WORKDIR /srv/

# Run the application
CMD ["markdownshare", "serve", "-host", "0.0.0.0", "-read-only" ]
