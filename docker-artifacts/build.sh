#!/bin/bash

go get -v ./...
go build -v
mv craft-config build/bin/craft-config_${GOOS}_${GOARCH}
