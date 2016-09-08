repo := craft-config-builder
aws_profile = momentlabs

prog := craft-config
release_dir := release
binaries := $(release_dir)/$(prog)_darwin_amd64 $(release_dir)/$(prog)_linux_amd64

help:
	@echo make release-build \# Creates the binaries: $(binaries)
	@echo make new-release version=v0.0.2 description="This is an early release." \# creates a release on github.
	@echo make release-publish version=v0.0.2 \# pushes the binaries to the github release.

clean:
	rm -rf release

# Only define these variables for the release build.
release-build: now := $(shell date +%s)
release-build: timeflag := -X $(prog)/version.unixtime=$(now)
release-build: hash := $(shell git rev-parse HEAD)
release-build: hashflag := -X $(prog)/version.githash=$(hash)
release-build: env := production
release-build: envflag := -X $(prog)/version.environ=$(env)
release-build: ld_args := $(envflag) $(hashflag) $(timeflag)

$(release_dir)/$(prog)_darwin_amd64 : 
	GOOS=darwin GOARC=amd64 go build "-ldflags=$(ld_args)" -o $(release_dir)/$(prog)_darwin_amd64

$(release_dir)/$(prog)_linux_amd64 : 
	GOOS=linux GOARC=amd64 go build "-ldflags=$(ld_args)" -o $(release_dir)/$(prog)_linux_amd64

release-build: $(binaries)
	@echo building with time: $(now) hash: $(hash) env: $(env)

# TODO: Consider doing some git tagging and building in a file for description.
new-release: clean release-build
	@echo creating release on github, version: ${version}: $(description)
	github-release release -u Momentlabs -r craft-config -t ${version} -d "${description}"
	github-release upload -u Momentlabs -r craft-config -t ${version} -n craft-config_linux_amd64 -f $(release_dir)/$(prog)_linux_amd64
	github-release upload -u Momentlabs -r craft-config -t ${version} -n craft-config_darwin_amd64 -f $(release_dir)/$(prog)_darwin_amd64

release-publish: release-build
	github-release upload -u Momentlabs -r craft-config -t ${version} -n craft-config_linux_amd64 -f $(release_dir)/$(prog)_linux_amd64
	github-release upload -u Momentlabs -r craft-config -t ${version} -n craft-config_darwin_amd64 -f $(release_dir)/$(prog)_darwin_amd64


