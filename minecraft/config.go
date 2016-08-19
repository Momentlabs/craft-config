package minecraft

import (
  "fmt"
  "os"
  // "io/ioutil"
  "github.com/go-ini/ini"
  // "errors"
  // "log"
)

func init() {
  ini.PrettyFormat = false
}
type ServerConfig struct {
  Config *ini.File
}

func defaultSection(config *ini.File) (*ini.Section) {
  section, err := config.GetSection("")
  checkFatalError(err)
  return section
}

func  NewConfigFromFile(fileName string) (*ServerConfig) {
  cfg, err := ini.Load(fileName)
  checkFatalError(err)

  return &ServerConfig{Config: cfg}
}

func (cfg *ServerConfig) WriteToFile(filename string) {
  file, err := os.Create(filename)
  checkFatalError(err)

  _, err = cfg.Config.WriteTo(file)
  checkFatalError(err)

  file.Close()
}

func (cfg *ServerConfig) SetEntry(key string, value string) {
  section  := defaultSection(cfg.Config)
  if section.HasKey(key) {
    entry, err := section.GetKey(key)
    checkFatalError(err)
    entry.SetValue(value)
  } else {
    log.Notice("Key \"%s\" not present in config. Configuration unmodified.", key)
  }
}

func (cfg *ServerConfig) HasKey(key string) (bool) {
  section := defaultSection(cfg.Config)
  return section.HasKey(key)
}

func (cfg *ServerConfig) List() {
  keyHash := defaultSection(cfg.Config).KeysHash()
  for key, value := range keyHash {
    fmt.Printf("%s: %s\n", key, value)
  }
}


