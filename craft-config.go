package main

import (
  "fmt"
  "gopkg.in/alecthomas/kingpin.v2"
  "os"
)

var (
  app                               *kingpin.Application
  verbose                           bool
  region                            string

  // Prompt for Commands
  interactive                       *kingpin.CmdClause

  serverConfig                      *kingpin.CmdClause
  listServerConfig                  *kingpin.CmdClause
  serverConfigFileName              string
)

func init() {
  app = kingpin.New("craft-config.go", "Command line to to manage minecraft server state.")
  app.Flag("verbose", "Describe what is happening, as it happens.").Short('v').BoolVar(&verbose)

  interactive = app.Command("interactive", "Prompt for commands.")

  serverConfig = app.Command("server-config", "Manage a server config.")
  listServerConfig = serverConfig.Command("list", "List out the server config")
  listServerConfig.Arg("server-config-file-name", "Name of the server config file").Required().StringVar(&serverConfigFileName)


  kingpin.CommandLine.Help = `A command-line minecraft config tool.`
}

func main() {

  // Parse the command line to fool with flags and get the command we'll execeute.
  command := kingpin.MustParse(app.Parse(os.Args[1:]))

   if verbose {
    fmt.Printf("Starting up.")
   }

   // This some state passed to each command (eg. an AWS session or connection)
   // So not usually a string.
   appContext := "AppContext"

  // List of commands as parsed matched against functions to execute the commands.
  commandMap := map[string]func(string) {
    listServerConfig.FullCommand(): doListServerConfig,
  }

  // Execute the command.
  if interactive.FullCommand() == command {
    doInteractive()
  } else {
    commandMap[command](appContext)
  }
}

func doListServerConfig(ctxt string) {
  serverConfig := newConfigFromFile(serverConfigFileName)
  for key, value := range *serverConfig.Config {
    fmt.Printf("%s: %s\n", key, value)
  }
}
