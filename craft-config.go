package main

import (
  "fmt"
  "github.com/alecthomas/kingpin"
  "os"
  "craft-config/interactive"
  "craft-config/lib"
  "craft-config/version"
  "github.com/aws/aws-sdk-go/aws/session"
  "github.com/jdrivas/sl"
  "github.com/Sirupsen/logrus"

  // "awslib"
  "github.com/jdrivas/awslib"
  // "mclib"
  "github.com/jdrivas/mclib"
)


var (
  DEFAULT_REGION = "us-west-1"
  DefaultBucket = "momentlabs-test"
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
  interactiveCmd                    *kingpin.CmdClause

  versionCmd                        *kingpin.CmdClause
  queryCmd                          *kingpin.CmdClause
  queryArg                          []string
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
  rconRetriesArg                    int
  rconDelayArg                      int
  publishArchiveArg                 bool
  serverIpArg                       string
  rconPortArg                       int64
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

  versionCmd = app.Command("version", "Print the version and exit.")
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
  archiveAndPublishCmd.Flag("rcon-port", "Port of server for rcon connection.").Default("25575").Int64Var(&rconPortArg)
  archiveAndPublishCmd.Flag("rcon-pw", "PW to connect ot the rcon server.").Default("testing").StringVar(&rconPasswordArg)
  archiveAndPublishCmd.Flag("rcon-retries", "Number of times to retry the connection before failure..").Default("-1").IntVar(&rconRetriesArg)
  archiveAndPublishCmd.Flag("rcon-delay", "Number of seconds to wait between retries..").Default("5").IntVar(&rconDelayArg)
  archiveAndPublishCmd.Flag("archive-directory","Where the server data is located.").Default(".").StringVar(&archiveDirectoryArg)
  archiveAndPublishCmd.Flag("bucket-name","S3 bucket for archive storage.").Default(DefaultBucket).StringVar(&bucketNameArg)
  archiveAndPublishCmd.Arg("user", "Name of user of the server were achiving.").StringVar(&userArg)
  archiveAndPublishCmd.Arg("server-name", "Name of the server were archiving.").StringVar(&serverNameArg)

  kingpin.CommandLine.Help = "A command-line minecraft config tool."

}

func main() {
  command := kingpin.MustParse(app.Parse(os.Args[1:]))
  if command == versionCmd.FullCommand() {
    fmt.Println(version.Version.String())
    os.Exit(0)
  }

  configureLogs()

  // Get the default session
  var sess *session.Session
  var err error

  //
  // Configure AWS for acrhive.
  //

  f := logrus.Fields{
    "controllerVersion": version.Version.String(),
    "profile": awsProfileArg, 
    "region": awsRegionArg,
  }

  // Note: We rely on NewSession to accomdate general defaults,
  // but, in particular, it supports auto EC2 IAMRole credential provisionsing.
  if awsProfileArg == "" {
    log.Debug(f, "Controller starting up: Getting AWS session with NewSession() and defaults.")
    sess, err = session.NewSession()
  } else {
    log.Debug(f, "Controller starting up: Getting AWS session with default with Profile.")
    sess, err = awslib.GetSession(awsProfileArg)
  }

  if err != nil {
    log.Error(logrus.Fields{"profile": awsProfileArg,}, 
      "Controller starting up: Can't get aws configuration information for session.", err)
  }

  accountAliases, err := awslib.GetAccountAliases(sess.Config)
  f["region"] = *sess.Config.Region
  if err == nil {
    f["account"] = accountAliases[0]
  } else {
    log.Error(f, "Craft-config startup: couldn't obtain account aliases.", err)
  }

  // Command line args trump the environment.
  userName := userArg
  serverName := serverNameArg
  archiveBucketName := bucketNameArg
  if userArg == "" && serverNameArg == "" { // get them from the env
    ServerUserKey := mclib.ServerUserKey
    ServerNameKey := mclib.ServerNameKey
    ArchiveBucketKey := mclib.ArchiveBucketKey
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
  clusterName := "<none>"
  if cn := os.Getenv(mclib.ClusterNameKey); cn != "" {
    clusterName = cn
  }
  taskArn := "<none>"
  if tn := os.Getenv(mclib.TaskArnKey); tn  != "" {
    taskArn = tn
  }



  // TODO: need to use some kind of Action function on the args.
  serverPort := mclib.Port(25565)
  rconPort := mclib.Port(25575)
  if rconPortArg != 0 {
    rconPort = mclib.Port(rconPortArg)
  }

  // List of commands as parsed matched against functions to execute the commands.
  // TODO: ClusterName and PrivateIP are not filled in here. This is probably not a problem
  // for current use ..... bad behaviour. Needs a fix, either command line or something else.
  server := &mclib.Server{
    User: userName,
    Name: serverName,
    ClusterName: clusterName,
    PublicServerIp: serverIpArg,
    // PrivateServerIp: 
    ServerPort: mclib.Port(serverPort),
    RconPort: mclib.Port(rconPort),
    RconPassword: rconPasswordArg,
    ArchiveBucket: archiveBucketName,
    TaskArn: &taskArn,
    ServerDirectory: archiveDirectoryArg,
    AWSSession: sess,
  }  
  f["userName"] = server.User
  f["serverName"] = server.Name
  f["bucketName"] = server.ArchiveBucket
  f["clusterName"] = server.ClusterName
  f["taskArn"] = server.TaskArn
  f["publicServerIp"] = server.PublicServerIp
  f["serverPort"] = server.ServerPort
  f["rconPort"] = server.RconPort
  log.Info(f, "Controller Start up complete.")


  commandMap := map[string]func(*mclib.Server) {
    listServerConfig.FullCommand(): doListServerConfig,
    modifyServerConfig.FullCommand(): doModifyServerConfig,
    archiveAndPublishCmd.FullCommand(): doArchiveAndPublish,
    queryCmd.FullCommand(): doQuery,
  }

  // Execute the command.
  if interactiveCmd.FullCommand() == command {
    // TODO: send along a server to the interactive UI as well.
    interactive.DoInteractive(debug, sess)
  } else {
    commandMap[command](server)
  }
}

//
// Command Implementations
//

func doQuery(server *mclib.Server) {
  lib.RconLoop(server.PublicServerIp, server.RconPort, server.RconPassword)
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

func bogusTest() (string) {
  return "hello"
}

func configureLogs() {
  setFormatter()
  updateLogLevel()
}


const (
  jsonLog = "json"
  textLog = "text"
)

func setFormatter() {
  var f logrus.Formatter
  switch logsFormatArg {
  case jsonLog:
    f = new(logrus.JSONFormatter)
  case textLog:
    s := new(sl.TextFormatter)
    s.FullTimestamp = true
    f = logrus.Formatter(s)
  }
  log.SetFormatter(f)
  mclib.SetLogFormatter(f)
  lib.SetLogFormatter(f)
  awslib.SetLogFormatter(f)
}

func updateLogLevel() {
  l := logrus.InfoLevel
  if debug || verbose {
    l = logrus.DebugLevel
  }
  log.SetLevel(l)
  mclib.SetLogLevel(l)
  lib.SetLogLevel(l)
  awslib.SetLogLevel(l)
}
