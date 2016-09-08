repo := craft-config-builder
aws_profile = momentlabs

prog := craft-config
release_dir := release
release_artifacts := $(release_dir)/$(prog)_darwin_amd64 $(release_dir)/$(prog)_linux_amd64

help:
	@echo make new-release version=v0.0.2 description="This is an early release." \# creates a release on github.
	@echo make release-publish verion=v0.0.2 \# pushes the binaries to the github release.

# Only define these variables for the release build.
release-build: now := $(shell date +%s)
release-build: timeflag := -X main.unixtime=$(now)
release-build: hash := $(shell git rev-parse HEAD)
release-build: hashflag := -X main.githash=$(hash)
release-build: env := production
release-build: envflag := -X main.environ=$(env)
release-build: ld_args := $(envflag) $(hasflag) $(timeflag)


release-build:
	GOOS=linux GOARC=amd64 go build "-ldflags=$(ld_args)" -o $(release_dir)/$(prog)_linux_amd64
	GOOS=darwin GOARC=amd64 go build "-ldflags=$(ld_args)" -o $(release_dir)/$(prog)_darwin_amd64

new-release:
	@echo creating release on github, version: ${version}: $(description)
	github-release release -u Momentlabs -r craft-config -t ${version} -d "${description}"

release-publish: $(release_artifacts)
	github-release upload -u Momentlabs -r craft-config -t ${version} -n craft-config_linux_amd64 -f build/bin/craft-config_linux_amd64
	github-release upload -u Momentlabs -r craft-config -t ${version} -n craft-config_darwin_amd64 -f build/bin/craft-config_darwin_amd64

