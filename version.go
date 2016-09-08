package main

import (
  "fmt"
  "strconv"
  "time"
  "github.com/jdrivas/mclib"  
)

// Version
// These will get set by ldFlags during the build.
var (
  // buildstamp string
  githash string
  environ string
  unixtime string
)

type  AppVersion struct {
    major int
    minor int
    dot int
    githash string
    environ string
    buildStamp time.Time
}

var Version AppVersion
func init() {
  ut, err := strconv.ParseInt(unixtime, 10, 64)
  if err != nil {
    ut = 0
  }
  buildTime := time.Unix(ut, 0)
  Version = AppVersion{
    major: 0,
    minor: 0,
    dot: 1,
    githash: githash,
    environ: environ,
    buildStamp: buildTime,
  }
}

func (AppVersion) String() string {
  return fmt.Sprintf("Version: %d.%d.%d (%s) %s [%s]\n", 
    Version.major, Version.minor, Version.dot,
    Version.environ, Version.buildStamp.Local().Format(time.RFC1123), Version.githash)
  // return fmt.Sprintf("Version: %d.%d.%d %s [%s] %s.\n", 
  //   Version.major, Version.minor, Version.dot,
  //   Version.environ, Version.githash, Version.buildstamp)
}

func doPrintVersion(*mclib.Server) {
  fmt.Printf("%s.\n", Version)
}