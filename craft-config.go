package main

import (
  "fmt"
  "gopkg.in/alecthomas/kingpin.v2"
  "os"
  "time"
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
  awsConfigFileArg                  string

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
  continuousArchiveArg              bool
  serverIpArg                       string
  rconPortArg                       string
  rconPassword                      string

  log                               *logging.Logger
  awsConfig                         *aws.Config
)


func init() {
  log = logging.MustGetLogger("craft-config")

  app = kingpin.New("craft-config.go", "Command line to to manage minecraft server state.")
  app.Flag("verbose", "Describe what is happening, as it happens.").BoolVar(&verbose)
  app.Flag("debug", "Set logging level to debug: lots of logging.").BoolVar(&debug)
  app.Flag("aws-config", "Configuration file location.").StringVar(&awsConfigFileArg)

  interactiveCmd = app.Command("interactive", "Prompt for commands.")

  serverConfig = app.Command("server-config", "Manage a server config.")

  listServerConfig = serverConfig.Command("list", "List out the server config")
  listServerConfig.Arg("server-config-file-name", "Name of the server config file").Required().StringVar(&serverConfigFileName)

  modifyServerConfig = serverConfig.Command("modify", "change a key value. Key must be present in source file.")
  modifyServerConfig.Arg("entries", "Key value pair configuration entries.").Required().StringMapVar(&keyValueMap)
  modifyServerConfig.Flag("source-file", "Source configuration to read.").Default("server.cfg").Short('s').StringVar(&serverConfigFileName)
  modifyServerConfig.Flag("dest-file", "Modified file to write. If not then new config goes to stdout.").Required().Short('d').StringVar(&newServerConfigFileName)

  archiveAndPublishCmd = app.Command("archive", "Archive a server and Publish archive to S3.")  
  archiveAndPublishCmd.Flag("continuous", "Continously archive and publish, when users are logged into the server.").BoolVar(&continuousArchiveArg)
  archiveAndPublishCmd.Flag("server-ip", "IP address for the rcon server connection.").Default("127.0.0.1").StringVar(&serverIpArg)
  archiveAndPublishCmd.Flag("rcon-port", "Port for rcon server connection.").Default("25575").StringVar(&rconPortArg)
  archiveAndPublishCmd.Flag("rcon-pw", "PW to connect ot the rcon server.").Default("testing").StringVar(&rconPassword)
  archiveAndPublishCmd.Arg("user", "Name of user for archive publishing.").Required().StringVar(&userArg)
  archiveAndPublishCmd.Arg("bucket name","S3 bucket for archive storage.").Default("craft-config-test").StringVar(&bucketNameArg)
  archiveAndPublishCmd.Arg("archive directory","to archive.a").Default(".").StringVar(&archiveDirectoryArg)

  kingpin.CommandLine.Help = "A command-line minecraft config tool."
}

func main() {

  keyValueMap = make(map[string]string)
  // Parse the command line to fool with flags and get the command we'll execeute.
  command := kingpin.MustParse(app.Parse(os.Args[1:]))
  setLogLevel(logging.INFO)
  if verbose {
    setLogLevel(logging.DEBUG)
  }
  if debug {
    setLogLevel(logging.DEBUG)
  }

  awsConfig = awslib.GetConfig("minecraft", awsConfigFileArg)
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
  var rcon *minecraft.Rcon
  var err error
  waitTime := 5 * time.Second
  count := 0
  for rcon == nil {
    rcon, err = minecraft.NewRcon(serverIpArg, rconPortArg, rconPassword)
    count++
    if err != nil { 
      log.Infof("Rcon creation failed: %s. Sleeping for %s.", err, waitTime)
      rcon = nil
    }
    if count > 10 { break }
    time.Sleep(waitTime)
  }
  if rcon == nil {
    log.Info("Failed to create an Rcon to the server. Can't archive.")
    return
  } else {
    log.Info("RCON Connected to server.")
  }

  if continuousArchiveArg {
    continuousArchiveAndPublish(rcon, archiveDirectoryArg, bucketNameArg, userArg, awsConfig)
  } else {
    archiveAndPublish(rcon, archiveDirectoryArg, bucketNameArg, userArg, awsConfig)
  }
}

func continuousArchiveAndPublish(rcon *minecraft.Rcon, archiveDir, bucketName, user string, cfg *aws.Config) {
  delayTime := 5 * time.Minute
  for {
    users, err := rcon.NumberOfUsers()
    if err != nil { 
      log.Errorf("Can't get the numbers of users from the server. %s", err)
      return
    } 
    if users > 0 {
      archiveAndPublish(rcon, archiveDirectoryArg, bucketNameArg, userArg, awsConfig)
    } else {
      log.Info("No users on server. Not updating the archive.")
    }
    time.Sleep(delayTime)
  }
}

func archiveAndPublish(rcon *minecraft.Rcon, archiveDir, bucketName, user string, cfg *aws.Config) {
  resp, err := minecraft.ArchiveAndPublish(rcon, archiveDir, bucketName, user, cfg)
  if err != nil {
    log.Errorf("Error creating an archive and publishing to S3: %s", err)
  } else {
    log.Infof("Published archive to: %s:%s\n", resp.BucketName, resp.StoredPath)
  }
}

func setLogLevel(level logging.Level) {
  logs := []string{"craft-config", "craft-config/interactive", "craft-config/minecraft"}
  for _, logName := range logs {
    log.Infof("Setting log \"%s\" to %s\n", logName, level)
    logging.SetLevel(level, logName)
  }
}

