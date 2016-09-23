package interactive

import(
  "fmt"
  "mclib"
  // "github.com/jdrivas/mclib"
)


func doReadServerConfigFile() (error) {
  currentServerConfig = mclib.NewConfigFromFile(currentServerConfigFileNameArg)
  return nil
}

func doPrintServerConfig() (error) {
  currentServerConfig.List()
  return nil
}

func doWriteServerConfig() (error) {
  if verbose {
    fmt.Printf("Writing out file: \"%s\"", newServerConfigFileNameArg)
  }
  currentServerConfig.WriteToFile(newServerConfigFileNameArg)
  return nil
}

func doSetServerConfigValue() (error) {
  currentServerConfig.SetEntry(currentKeyArg, currentValueArg)
  return nil
}