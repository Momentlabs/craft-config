FROM golang

MAINTAINER momentlabs

VOLUME ["/go/src/craft-config"]

COPY docker-artifacts/build.sh /build
RUN chmod +x /build
WORKDIR "/go/src/craft-config"

ENV GOOS=linux GOARCH=amd64

ENTRYPOINT ["/build"]

