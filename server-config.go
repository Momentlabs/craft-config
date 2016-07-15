package main

import (
  // "fmt"
  // "os"
  // "io/ioutil"
  "github.com/go-ini/ini"
  // "errors"
  // "log"
)

type ServerConfig struct {
  Config *map[string]string
}

func  newConfigFromFile(fileName string) (*ServerConfig) {
  cfg, err := ini.Load(fileName)
  checkFatalError(err)

  section, err := cfg.GetSection("")
  checkFatalError(err)

  configHash := section.KeysHash()

  return &ServerConfig{Config: &configHash}
}


