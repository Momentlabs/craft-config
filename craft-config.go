package main

import (
  "fmt"
  "github.com/alecthomas/kingpin"
  "os"
  "time"
  "ecs-pilot/awslib"
  "craft-config/interactive"
  // "craft-config/minecraft"
  "github.com/aws/aws-sdk-go/aws"
  "github.com/jdrivas/mclib"
  "github.com/jdrivas/sl"
  "github.com/Sirupsen/logrus"
)

// Log formats.
const (
  jsonLog = "json"
  textLog = "text"
)

var (
  app                               *kingpin.Application
  verbose                           bool
  debug                             bool
  region                            string
  awsConfigFileArg                  string
  profileArg                        string
  logsFormatArg                     string

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
  serverNameArg                     string
  bucketNameArg                     string
  archiveDirectoryArg               string
  continuousArchiveArg              bool
  useRconArg                        bool
  publishArchiveArg                 bool
  serverIpArg                       string
  rconPortArg                       string
  rconPassword                      string

  log = sl.New()
  awsConfig                         *aws.Config
)


func init() {

  keyValueMap = make(map[string]string)

  app = kingpin.New("craft-config.go", "Command line to to manage minecraft server state.")
  app.Flag("verbose", "Describe what is happening, as it happens.").BoolVar(&verbose)
  app.Flag("debug", "Set logging level to debug: lots of logging.").BoolVar(&debug)
  app.Flag("aws-config", "Configuration file location.").StringVar(&awsConfigFileArg)
  app.Flag("log-format", "Choose text or json output.").Default(jsonLog).EnumVar(&logsFormatArg, jsonLog, textLog)
  app.Flag("profile", "AWS profile for credentials.").Default("minecraft").StringVar(&profileArg)

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
  archiveAndPublishCmd.Flag("noPublish", "Don't publish the archive to S3, just create it.").Default("true").BoolVar(&publishArchiveArg)
  archiveAndPublishCmd.Flag("noRcon", "Don't try to use the RCON connection on the server to start/stop saving.  UNSAFE").Default("true").BoolVar(&useRconArg)
  archiveAndPublishCmd.Flag("rcon-port", "Port of server for rcon connection.").Default("25575").StringVar(&rconPortArg)
  archiveAndPublishCmd.Flag("rcon-pw", "PW to connect ot the rcon server.").Default("testing").StringVar(&rconPassword)
  archiveAndPublishCmd.Flag("archive-directory","Where the server data is located.").Default(".").StringVar(&archiveDirectoryArg)
  archiveAndPublishCmd.Flag("bucket-name","S3 bucket for archive storage.").Default("craft-config-test").StringVar(&bucketNameArg)
  archiveAndPublishCmd.Arg("user", "Name of user of the server were achiving.").Required().StringVar(&userArg)
  archiveAndPublishCmd.Arg("server-name", "Name of the server were archiving.").Default("TestServer").StringVar(&serverNameArg)

  kingpin.CommandLine.Help = "A command-line minecraft config tool."
}

func main() {
  command := kingpin.MustParse(app.Parse(os.Args[1:]))
  configureLogs()

  awsConfig = awslib.GetConfig(profileArg, awsConfigFileArg)
  region = *awsConfig.Region
  accountAliases, err := awslib.GetAccountAliases(awsConfig)
  if err == nil {
    log.Debug(logrus.Fields{"account": *accountAliases[0], "region": region}, "craft-config startup.")
  }

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


//
// Command Implementations
//

func doListServerConfig() {
  serverConfig := mclib.NewConfigFromFile(serverConfigFileName)
  serverConfig.List()
}

func doModifyServerConfig() {
  serverConfig := mclib.NewConfigFromFile(serverConfigFileName)
  for k, v := range keyValueMap {
    if verbose {fmt.Printf("Modifying: \"%s\" = \"%s\"\n", k, v)}
    if serverConfig.HasKey(k) {
      serverConfig.SetEntry(k,v)
      serverConfig.WriteToFile(newServerConfigFileName)
    } else {
      log.Fatal(logrus.Fields{"key": k,"config-file": serverConfigFileName,}, "Key not found in configuration. No files updated", nil)
    }
  }
}

func doArchiveAndPublish() {
  retries := 10
  waitTime := 5 * time.Second
  server := mclib.NewServer(userArg, serverNameArg, serverIpArg, rconPortArg, rconPassword, bucketNameArg, archiveDirectoryArg, awsConfig)
  server.NewRconWithRetry(retries, waitTime)
  if continuousArchiveArg {
    continuousArchiveAndPublish(server)
  } else {
    archiveAndPublish(server)
  }
}

// TODO: Set up some asynchronous go routines:
// 1. Delay timer: every 5 mniutes or so, come along and do a backup if there are users (what we have now).
// 2. File Watcher: check to see if non-world files have been created and update those.
// 3. User Watcher: assuming the file watcher, can't proxy for this. Set a timeout for every 10 seconds and check for new users,
// update when one shows up.
//
// Finally  Put the whole thing in a go-routine that checks for a stop (see the watcher in ineteractive.)
func continuousArchiveAndPublish(s *mclib.Server) {
  delayTime := 5 * time.Minute
  for {
    users, err := s.Rcon.NumberOfUsers()
    if err != nil { 
      log.Error(nil, "Can't get the numbers of users from the server.", err)
      return
    } 
    if users > 0 {
      archiveAndPublish(s)
    } else {
      log.Info(logrus.Fields{"retryDelay": delayTime.String(),}, "No users on server. Not updating the archive.")
    }
    time.Sleep(delayTime)
  }
}

func archiveAndPublish(s *mclib.Server) {

  resp, err := s.SnapshotAndPublish()

  archiveFields := logrus.Fields{
    "user": s.User,
    "serverName": s.Name,
    "serverDir": s.ServerDirectory,
    "bucket": s.ArchiveBucket,
  }
  if err != nil {
    log.Error(archiveFields, "Error creating an archive and publishing to S3.", err)
  } else {
    archiveFields["bucket"] = resp.BucketName
    archiveFields["archive"] = resp.StoredPath
    log.Info(archiveFields, "Published archive.")
  }
}

// func archiveAndPublish(rcon *mclib.Rcon, archiveDir, bucketName, user string, cfg *aws.Config) {
//   archiveFields := logrus.Fields{"archiveDir": archiveDir,"bucket": bucketName, "user": user, }
//   resp, err := mclib.ArchiveAndPublish(rcon, archiveDir, bucketName, user, cfg)
//   if err != nil {
//     log.Error(archiveFields, "Error creating an archive and publishing to S3.", err)
//   } else {
//     archiveFields["bucket"] = resp.BucketName
//     archiveFields["archive"] = resp.StoredPath
//     log.Info(archiveFields, "Published archive.")
//   }
// }


func configureLogs() {
  setFormatter()
  updateLogLevel()
}

  // TODO: Clearly this should set a *formatter in the switch
  // and then set each of the loggers to that formater.
  // The foramtters are of different types though, and 
  // lorgus.Formatter is an interface so I can't create a 
  // var out of it. I konw there is a way, I just don't know what it is.func setFormatter() {
func setFormatter() {
  switch logsFormatArg {
  case jsonLog:
    f := new(logrus.JSONFormatter)
    log.SetFormatter(f)
    mclib.SetLogFormatter(f)
  case textLog:
    f := new(sl.TextFormatter)
    f.FullTimestamp = true
    log.SetFormatter(f)
    mclib.SetLogFormatter(f)
  }
}

func updateLogLevel() {
  l := logrus.InfoLevel
  if debug || verbose {
    l = logrus.DebugLevel
  }
  log.SetLevel(l)
  mclib.SetLogLevel(l)
}

