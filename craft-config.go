package main

import (
  "fmt"
  "gopkg.in/alecthomas/kingpin.v2"
  "os"
  "log"
  "ecs-pilot/awslib"
  "craft-config/interactive"
  "craft-config/minecraft"
  // "github.com/aws/aws-sdk-go/aws"
  "github.com/op/go-logging"
)

var (
  app                               *kingpin.Application
  verbose                           bool
  region                            string

  // Prompt for Commands
  interactiveCmd                       *kingpin.CmdClause

  serverConfig                      *kingpin.CmdClause
  listServerConfig                  *kingpin.CmdClause
  serverConfigFileName              string
  modifyServerConfig                *kingpin.CmdClause
  newServerConfigFileName           string
  keyValueMap                       map[string]string
)

func init() {
  app = kingpin.New("craft-config.go", "Command line to to manage minecraft server state.")
  app.Flag("verbose", "Describe what is happening, as it happens.").Short('v').BoolVar(&verbose)

  interactiveCmd = app.Command("interactive", "Prompt for commands.")

  serverConfig = app.Command("server-config", "Manage a server config.")

  listServerConfig = serverConfig.Command("list", "List out the server config")
  listServerConfig.Arg("server-config-file-name", "Name of the server config file").Required().StringVar(&serverConfigFileName)

  modifyServerConfig = serverConfig.Command("modify", "change a key value. Key must be present in source file.")
  modifyServerConfig.Arg("entries", "Key value pair configuration entries.").Required().StringMapVar(&keyValueMap)
  modifyServerConfig.Flag("source-file", "Source configuration to read.").Default("server.cfg").Short('s').StringVar(&serverConfigFileName)
  modifyServerConfig.Flag("dest-file", "Modified file to write. If not then new config goes to stdout.").Required().Short('d').StringVar(&newServerConfigFileName)

  kingpin.CommandLine.Help = `A command-line minecraft config tool.`
}

func main() {

  keyValueMap = make(map[string]string)
  // Parse the command line to fool with flags and get the command we'll execeute.
  command := kingpin.MustParse(app.Parse(os.Args[1:]))
  if verbose {
    logging.SetLevel(logging.DEBUG, "craft-config/minecraft")
  }


  awsConfig := awslib.GetConfig("minecraft")
  fmt.Printf("%s\n", awslib.AccountDetailsString(awsConfig))

  // List of commands as parsed matched against functions to execute the commands.
  commandMap := map[string]func() {
    listServerConfig.FullCommand(): doListServerConfig,
    modifyServerConfig.FullCommand(): doModifyServerConfig,
  }

  // Execute the command.
  if interactiveCmd.FullCommand() == command {
    interactive.DoInteractive(awsConfig)
  } else {
    commandMap[command]()
  }
}

func doListServerConfig() {
  serverConfig := minecraft.NewConfigFromFile(serverConfigFileName)
  serverConfig.List()
}

func doModifyServerConfig() {
  serverConfig := minecraft.NewConfigFromFile(serverConfigFileName)
  for k, v := range keyValueMap {
    if verbose {fmt.Printf("Modifying: \"%s\" = \"%s\"\n", k, v)}
    if serverConfig.HasKey(k) {
      serverConfig.SetEntry(k,v)
      serverConfig.WriteToFile(newServerConfigFileName)
    } else {
      log.Fatalf("Key \"%s\" not found in configuration \"%s\". No files updated.\n",k,serverConfigFileName)
    }
  }
}
