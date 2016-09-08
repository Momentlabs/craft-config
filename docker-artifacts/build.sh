#!/bin/bash
env=production

# I hate this shit.
# now=`date  "+%A, %d %Y %H:%M %Z"`
now=`date +%s`
timeflag="-X main.unixtime=${now}"
hash=`git rev-parse HEAD`
hashflag="-X main.githash=${hash}"
envflag="-X main.environ=${env}"

args="${envflag} ${hashflag} ${timeflag}"

echo timeflag ${timeflag}
echo hashflag ${hashflag}
echo envflag ${envflag}
echo args ${args}

go get -v ./...
go build -v "-ldflags=${args}"
mv craft-config build/bin/craft-config_${GOOS}_${GOARCH}
