package main

import (
  "fmt"
  "gopkg.in/alecthomas/kingpin.v2"
  "os"
  // "log"
  "ecs-pilot/awslib"
  "craft-config/interactive"
  "craft-config/minecraft"
  "github.com/aws/aws-sdk-go/aws"
  "github.com/op/go-logging"
)

var (
  app                               *kingpin.Application
  verbose                           bool
  debug                             bool
  region                            string

  // Prompt for Commands
  interactiveCmd                       *kingpin.CmdClause

  serverConfig                      *kingpin.CmdClause
  listServerConfig                  *kingpin.CmdClause
  serverConfigFileName              string
  modifyServerConfig                *kingpin.CmdClause
  newServerConfigFileName           string
  keyValueMap                       map[string]string

  archiveAndPublishCmd              *kingpin.CmdClause
  userArg                           string
  bucketNameArg                     string
  archiveDirectoryArg               string

  log = logging.MustGetLogger("craft-config")
  awsConfig                         *aws.Config

)

func init() {
  logging.SetLevel(logging.INFO, "craft-config")

  app = kingpin.New("craft-config.go", "Command line to to manage minecraft server state.")
  app.Flag("verbose", "Describe what is happening, as it happens.").Short('v').BoolVar(&verbose)
  app.Flag("debug", "Set logging level to debug: lots of loggin.").Short('d').BoolVar(&debug)

  interactiveCmd = app.Command("interactive", "Prompt for commands.")

  serverConfig = app.Command("server-config", "Manage a server config.")

  listServerConfig = serverConfig.Command("list", "List out the server config")
  listServerConfig.Arg("server-config-file-name", "Name of the server config file").Required().StringVar(&serverConfigFileName)

  modifyServerConfig = serverConfig.Command("modify", "change a key value. Key must be present in source file.")
  modifyServerConfig.Arg("entries", "Key value pair configuration entries.").Required().StringMapVar(&keyValueMap)
  modifyServerConfig.Flag("source-file", "Source configuration to read.").Default("server.cfg").Short('s').StringVar(&serverConfigFileName)
  modifyServerConfig.Flag("dest-file", "Modified file to write. If not then new config goes to stdout.").Required().Short('d').StringVar(&newServerConfigFileName)

  archiveAndPublishCmd = app.Command("archive", "Archive a server and Publish archive to S3.")  
  archiveAndPublishCmd.Arg("user", "Name of user for archive publishing.").Required().StringVar(&userArg)
  archiveAndPublishCmd.Arg("bucket name","S3 bucket for archive storage.").Default("craft-config-test").StringVar(&bucketNameArg)
  archiveAndPublishCmd.Arg("archive directory","to archive.a").Default(".").StringVar(&archiveDirectoryArg)

  kingpin.CommandLine.Help = `A command-line minecraft config tool.`
}

func main() {

  keyValueMap = make(map[string]string)
  // Parse the command line to fool with flags and get the command we'll execeute.
  command := kingpin.MustParse(app.Parse(os.Args[1:]))
  setLogLevel(logging.WARNING)
  if verbose {
    setLogLevel(logging.INFO)
  }
  if debug {
    setLogLevel(logging.DEBUG)
  }

  awsConfig = awslib.GetConfig("minecraft")
  fmt.Printf("%s\n", awslib.AccountDetailsString(awsConfig))

  // List of commands as parsed matched against functions to execute the commands.
  commandMap := map[string]func() {
    listServerConfig.FullCommand(): doListServerConfig,
    modifyServerConfig.FullCommand(): doModifyServerConfig,
    archiveAndPublishCmd.FullCommand(): doArchiveAndPublish,
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

func doArchiveAndPublish() {
  rcon, err := minecraft.NewRcon("127.0.0.1", "25575", "testing")
  if err != nil { log.Infof("Rcon creation failed: %s", err) }
  resp, err := minecraft.ArchiveAndPublish(rcon, archiveDirectoryArg, bucketNameArg, userArg, awsConfig)
  if err != nil {
    log.Errorf("Error creating an archive and publishing to S3: %s", err)
  }
  log.Infof("Published archive to: %s:%s\n", resp.BucketName, resp.StoredPath)
}

func setLogLevel(level logging.Level) {
  logs := []string{"craft-config", "craft-config/interactive", "craft-config/minecraft"}
  for _, log := range logs {
    logging.SetLevel(level, log)
  }
}

