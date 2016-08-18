package interactive 

import (
  "gopkg.in/alecthomas/kingpin.v2"
  "github.com/bobappleyard/readline"
  "strings"
  "fmt"
  "io"
  "craft-config/minecraft"
)

var (

  interApp *kingpin.Application

  interExit *kingpin.CmdClause
  interQuit *kingpin.CmdClause
  interVerbose *kingpin.CmdClause
  iVerbose bool
  interTestString []string


  // Read a configureation file in the current config
  currentServerConfigFileName string
  currentServerConfig *minecraft.ServerConfig
  interReadServerConfigFile *kingpin.CmdClause

  // Print the current configuration out.
  interPrintServerConfig *kingpin.CmdClause

  // Write the current configuraiton out.
  interNewServerConfigFileName string
  interWriteServerConfig *kingpin.CmdClause

  // Set a key value, key must already be present.
  interSetServerConfigValue *kingpin.CmdClause
  currentKey string
  currentValue string

)

func init() {
  interApp = kingpin.New("", "Interactive mode.").Terminate(doTerminate)

  // state
  interVerbose = interApp.Command("verbose", "toggle verbose mode.")
  interExit = interApp.Command("exit", "exit the program. <ctrl-D> works too.")
  interQuit = interApp.Command("quit", "exit the program.")

  // Read and manipulate a configuration file.
  interReadServerConfigFile = interApp.Command("read-config", "read a server config file in.")
  interReadServerConfigFile.Arg("file-name", "The file to read the configuration file from.").Required().StringVar(&currentServerConfigFileName)

  interPrintServerConfig = interApp.Command("print-config", "print the server config file.")

  interWriteServerConfig = interApp.Command("write-config", "write the server config file.")
  interWriteServerConfig.Arg("file-name", "The file to write the confiugration file to.").Required().StringVar(&interNewServerConfigFileName)

  interSetServerConfigValue = interApp.Command("set-config-value", "set a configuration value - key must already be present.")
  interSetServerConfigValue.Arg("key", "Key for the setting - must be already presetn int he configuration").Required().StringVar(&currentKey)
  interSetServerConfigValue.Arg("value", "Value for the setting.").Required().StringVar(&currentValue)
}


func doICommand(line string, ctxt string) (err error) {

  // This is due to a 'peculiarity' of kingpin: it collects strings as arguments across parses.
  interTestString = []string{}

  // Prepare a line for parsing
  line = strings.TrimRight(line, "\n")
  fields := []string{}
  fields = append(fields, strings.Fields(line)...)
  if len(fields) <= 0 {
    return nil
  }

  command, err := interApp.Parse(fields)
  if err != nil {
    fmt.Printf("Command error: %s.\nType help for a list of commands.\n", err)
    return nil
  } else {
    switch command {
      case interVerbose.FullCommand(): err = doVerbose()
      case interExit.FullCommand(): err = doQuit()
      case interQuit.FullCommand(): err = doQuit()
      // case interTest.FullCommand(): err = doTest()
      case interReadServerConfigFile.FullCommand(): err = doReadServerConfigFile()
      case interPrintServerConfig.FullCommand(): err = doPrintServerConfig()
      case interWriteServerConfig.FullCommand(): err = doWriteServerConfig()
      case interSetServerConfigValue.FullCommand(): err = doSetServerConfigValue()
    }
  }
  return err
}

// Interactive Command processing
func doReadServerConfigFile() (error) {
  currentServerConfig = minecraft.NewConfigFromFile(currentServerConfigFileName)
  return nil
}

func doPrintServerConfig() (error) {
  currentServerConfig.List()
  return nil
}

func doWriteServerConfig() (error) {
  if iVerbose {
    fmt.Printf("Writing out file: \"%s\"", interNewServerConfigFileName)
  }
  currentServerConfig.WriteToFile(interNewServerConfigFileName)
  return nil
}

func doSetServerConfigValue() (error) {
  currentServerConfig.SetEntry(currentKey, currentValue)
  return nil
}


// Interactive Mode support functions.
func toggleVerbose() bool {
  iVerbose = !iVerbose
  return iVerbose
}

func doVerbose() (error) {
  if toggleVerbose() {
    fmt.Println("Verbose is on.")
  } else {
    fmt.Println("Verbose is off.")
  }
  return nil
}

func doQuit() (error) {
  return io.EOF
}

func doTerminate(i int) {}

func promptLoop(prompt string, process func(string) (error)) (err error) {
  errStr := "Error - %s.\n"
  for moreCommands := true; moreCommands; {
    line, err := readline.String(prompt)
    if err == io.EOF {
      moreCommands = false
    } else if err != nil {
      fmt.Printf(errStr, err)
    } else {
      readline.AddHistory(line)
      err = process(line)
      if err == io.EOF {
        moreCommands = false
      } else if err != nil {
        fmt.Printf(errStr, err)
      }
    }
  }
  return nil
}

// This gets called from the main program, presumably from the 'interactive' command on main's command line.
func DoInteractive() {
  xICommand := func(line string) (err error) {return doICommand(line, "craft-config")}
  prompt := "> "
  err := promptLoop(prompt, xICommand)
  if err != nil {fmt.Printf("Error - %s.\n", err)}
}




