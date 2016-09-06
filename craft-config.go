package main

import (
  "fmt"
  "github.com/alecthomas/kingpin"
  "os"
  "time"
  "craft-config/interactive"
  // "craft-config/minecraft"
  // "github.com/aws/aws-sdk-go/aws"
  "github.com/aws/aws-sdk-go/aws/session"
  "github.com/jdrivas/sl"
  "github.com/Sirupsen/logrus"

  // THIS IS FOR DEVELOPMENT PURPOSES AND
  // WILL LIKELY CAUSE ME TROUBLE.
  // PROBABLY BEST TO REMOVE THIS ONCE 
  // THE LIBRARIES ARE STABLE.
  // "mclib"
  "github.com/jdrivas/awslib"
  "github.com/jdrivas/mclib"
)



var (
  DEFAULT_REGION = "us-west-1"
)

var (
  app                               *kingpin.Application
  verbose                           bool
  debug                             bool
  awsRegionArg                      string
  awsProfileArg                     string
  // awsConfigFileArg                  string
  logsFormatArg                     string

  // Prompt for Commands
  interactiveCmd                       *kingpin.CmdClause

  queryCmd                      *kingpin.CmdClause
  queryArg                   []string
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
  rconPasswordArg                   string

  log = sl.New()
  sess *session.Session
)


func init() {

  keyValueMap = make(map[string]string)

  app = kingpin.New("craft-config.go", "Command line to to manage minecraft server state.")
  app.Flag("verbose", "Describe what is happening, as it happens.").BoolVar(&verbose)
  app.Flag("debug", "Set logging level to debug: lots of logging.").BoolVar(&debug)

  app.Flag("log-format", "Choose text or json output.").Default(jsonLog).EnumVar(&logsFormatArg, jsonLog, textLog)

  // app.Flag("aws-config", "Configuration file location.").StringVar(&awsConfigFileArg)
  app.Flag("region", "Aws region to use as a default (publishing archives.)").StringVar(&awsRegionArg)
  app.Flag("profile", "AWS profile for configuration.").StringVar(&awsProfileArg)

  interactiveCmd = app.Command("interactive", "Prompt for commands.")

  queryCmd = app.Command("query", "Issues a command to the RCON port of a server.")
  queryCmd.Arg("query-command", "command string to the server.").Required().StringsVar(&queryArg)
  queryCmd.Flag("server-ip", "IP address of server to connect with.").Default("127.0.0.1").StringVar(&serverIpArg)
  queryCmd.Flag("rcon-pw", "Password for rcon").Default("testing").StringVar(&rconPasswordArg)

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
  archiveAndPublishCmd.Flag("rcon-pw", "PW to connect ot the rcon server.").Default("testing").StringVar(&rconPasswordArg)
  archiveAndPublishCmd.Flag("archive-directory","Where the server data is located.").Default(".").StringVar(&archiveDirectoryArg)
  archiveAndPublishCmd.Flag("bucket-name","S3 bucket for archive storage.").Default("craft-config-test").StringVar(&bucketNameArg)
  archiveAndPublishCmd.Arg("user", "Name of user of the server were achiving.").StringVar(&userArg)
  archiveAndPublishCmd.Arg("server-name", "Name of the server were archiving.").StringVar(&serverNameArg)

  kingpin.CommandLine.Help = "A command-line minecraft config tool."
}

func main() {
  command := kingpin.MustParse(app.Parse(os.Args[1:]))

  configureLogs()


  // Get the default session
  var sess *session.Session
  var err error


  //
  // Configure AWS for acrhive.
  //

  // Note: We rely on NewSession to accomdate general defaults,
  // but, in particular, it supports auto EC2 IAMRole credential provisionsing.
  f := logrus.Fields{"profile": awsProfileArg, "region": awsRegionArg,}

  if awsProfileArg == "" {
    log.Debug(f, "Getting AWS session with NewSession() and defaults.")
    sess, err = session.NewSession()
  } else {
    log.Debug(f, "Getting AWS session with default with Profile.")
    sess, err = awslib.GetSession(awsProfileArg, "")
  }

  if err != nil {
    log.Error(logrus.Fields{"profile": awsProfileArg,}, 
      "Can't get aws configuration information for session.", err)
  }

  accountAliases, err := awslib.GetAccountAliases(sess.Config)
  if err == nil {
    log.Debug(logrus.Fields{"account": *accountAliases[0], "region": *sess.Config.Region}, "craft-config startup.")
  }

  // Command line args trump the environment.
  userName := userArg
  serverName := serverNameArg
  archiveBucketName := bucketNameArg
  if userArg == "" && serverNameArg == "" { // get them from the env
    // TODO: THESE ARE ACTUALLY DEFINED IN ecs-craft. The clearly need
    // to be moved somewhere else, probably mclib.
    ServerUserKey := "SERVER_USER"
    ServerNameKey := "SERVER_NAME"
    ArchiveBucketKey := "ARCHIVE_BUCKET"
    if u := os.Getenv(ServerUserKey); u != "" {
      userName = u
    }
    if s := os.Getenv(ServerNameKey); s != "" {
      serverName = s
    }
    if b := os.Getenv(ArchiveBucketKey); b != "" {
      archiveBucketName = b
    }
  }
  log.Debug(logrus.Fields{
    "userName": userName, 
    "serverName": serverName, 
    "bucketName": archiveBucketName,
  }, "Got user, server and bucket names.")

  // TODO: This has to change ....
  serverPort := int64(25565)
  rconPort := "25575"
  if rconPortArg != "" {
    rconPort = rconPortArg
  }

  // List of commands as parsed matched against functions to execute the commands.
  server := mclib.NewServer(userName, serverName, 
    serverIpArg, serverPort, rconPort, rconPasswordArg, archiveBucketName, archiveDirectoryArg, sess)

  commandMap := map[string]func(*mclib.Server) {
    listServerConfig.FullCommand(): doListServerConfig,
    modifyServerConfig.FullCommand(): doModifyServerConfig,
    archiveAndPublishCmd.FullCommand(): doArchiveAndPublish,
    queryCmd.FullCommand(): doQuery,
  }

  // Execute the command.
  if interactiveCmd.FullCommand() == command {
    // TODO: send along a server to the interactive UI as well.
    interactive.DoInteractive(sess)
  } else {
    commandMap[command](server)
  }
}

//
// Command Implementations
//


func doQuery(server *mclib.Server) {
  q := ""
  for _, e := range queryArg {
    q += e + " "
  }

  rc, err  := server.NewRcon()
  if err != nil {
    fmt.Printf("Error getting rcon connection to server: %s.\n", err)
  return
  }

  reply, err := rc.List()
  if err ==  nil {
    fmt.Printf("%s\n", reply)
  } else {
    fmt.Printf("Error with server command: %s.\n", err)
  }
}

func doListServerConfig(*mclib.Server) {
  serverConfig := mclib.NewConfigFromFile(serverConfigFileName)
  serverConfig.List()
}

func doModifyServerConfig(*mclib.Server) {
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


func doArchiveAndPublish(server *mclib.Server) {
  // server := mclib.NewServer(userArg, serverNameArg, serverIpArg, rconPortArg, rconPassword, bucketNameArg, archiveDirectoryArg, sess)
  retries := 10
  waitTime := 5 * time.Second
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

func bogusTest() (string) {
  return "hello"
}

func configureLogs() {
  setFormatter()
  updateLogLevel()
}

// TODO: Clearly this should set a *formatter in the switch
// and then set each of the loggers to that formater.
// The foramtters are of different types though, and 
// lorgus.Formatter is an interface so I can't create a 
// var out of it. I konw there is a way, I just don't know what it is.func setFormatter() {
// Log formats.
const (
  jsonLog = "json"
  textLog = "text"
)
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

