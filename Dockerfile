FROM golang

MAINTAINER David Rivas david@momentlabs.io

VOLUME ["/go/src/"]

# This is what the parent expects.
WORKDIR /go/src/craft-config

ENTRYPOINT ["make", "release/craft-config_linux_amd64"]
