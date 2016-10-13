repo := craft-config-builder
aws_profile = momentlabs

prog := craft-config
release_dir := release
builds := darwin_build linux_build
darwin_target := $(release_dir)/$(prog)_darwin_amd64
linux_target := $(release_dir)/$(prog)_linux_amd64 
# binaries := $(release_dir)/$(prog)_darwin_amd64 $(release_dir)/$(prog)_linux_amd64

.PHONY: check-env

help:
	@echo make release-build \# Creates the binaries: $(binaries)
	@echo make new-release version=v0.0.2 description="This is an early release." \# creates a release on github.
	@echo make publish-release version=v0.0.2 \# pushes the binaries to the github release.
	@echo Must define GITHUB_TOKEN to use the release commands.

clean:
	rm -rf release

# Only define these variables for the release build.
$(darwin_target) $(linux_target) : now := $(shell date +%s)
$(darwin_target) $(linux_target) : timeflag := -X $(prog)/version.unixtime=$(now)
$(darwin_target) $(linux_target) : hash := $(shell git rev-parse HEAD)
$(darwin_target) $(linux_target) : hashflag := -X $(prog)/version.githash=$(hash)
$(darwin_target) $(linux_target) : env := production
$(darwin_target) $(linux_target) : envflag := -X $(prog)/version.environ=$(env)
$(darwin_target) $(linux_target) : ld_args := $(envflag) $(hashflag) $(timeflag)

$(darwin_target) :
	GOOS=darwin GOARC=amd64 go build "-ldflags=$(ld_args)" -o $(release_dir)/$(prog)_darwin_amd64

$(linux_target) :
	GOOS=linux GOARC=amd64 go build "-ldflags=$(ld_args)" -o $(release_dir)/$(prog)_linux_amd64

darwin_build : $(darwin_target)

# This is a docker build to get a linux target because of golang cgo dependency in os.user
linux_build :
	docker-compose up --force-recreate 

release-build: $(builds)

# TODO: Consider doing some git tagging and building in a file for description.
# TODO: Note that this doesn't guarantee tha the source in the repo and the binary
# match. It relies on there already being a git commit and push.
new-release: clean release-build check-env
	@echo creating release on github, version: ${version}: $(description)
	github-release release -u Momentlabs -r craft-config -t ${version} -d "${description}"
	github-release upload -u Momentlabs -r craft-config -t ${version} -n craft-config_linux_amd64 -f $(release_dir)/$(prog)_linux_amd64
	github-release upload -u Momentlabs -r craft-config -t ${version} -n craft-config_darwin_amd64 -f $(release_dir)/$(prog)_darwin_amd64

publish-release: release-build check_env
	github-release upload -u Momentlabs -r craft-config -t ${version} -n craft-config_linux_amd64 -f $(release_dir)/$(prog)_linux_amd64
	github-release upload -u Momentlabs -r craft-config -t ${version} -n craft-config_darwin_amd64 -f $(release_dir)/$(prog)_darwin_amd64

check-env:
ifndef GITHUB_TOKEN
	$(error GITHUB_TOKEN is undefined.)
endif
