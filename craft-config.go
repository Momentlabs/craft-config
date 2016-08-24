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
  "github.com/Sirupsen/logrus"
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
  useRconArg                        bool
  publishArchiveArg                 bool
  serverIpArg                       string
  rconPortArg                       string
  rconPassword                      string

  log = logrus.New()
  awsConfig                         *aws.Config
)


func init() {

  keyValueMap = make(map[string]string)

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
  archiveAndPublishCmd.Flag("noPublish", "Don't publish the archive to S3, just create it.").Default("true").BoolVar(&publishArchiveArg)
  archiveAndPublishCmd.Flag("noRcon", "Don't try to use the RCON connection on the server to start/stop saving.  UNSAFE").Default("true").BoolVar(&useRconArg)
  archiveAndPublishCmd.Flag("rcon-port", "Port for rcon server connection.").Default("25575").StringVar(&rconPortArg)
  archiveAndPublishCmd.Flag("rcon-pw", "PW to connect ot the rcon server.").Default("testing").StringVar(&rconPassword)
  archiveAndPublishCmd.Arg("user", "Name of user for archive publishing.").Required().StringVar(&userArg)
  archiveAndPublishCmd.Arg("bucket name","S3 bucket for archive storage.").Default("craft-config-test").StringVar(&bucketNameArg)
  archiveAndPublishCmd.Arg("archive directory","to archive.a").Default(".").StringVar(&archiveDirectoryArg)

  kingpin.CommandLine.Help = "A command-line minecraft config tool."
}

func main() {
  configureLogs()

  command := kingpin.MustParse(app.Parse(os.Args[1:]))
  updateLogLevel()

  awsConfig = awslib.GetConfig("minecraft", awsConfigFileArg)
  region = *awsConfig.Region
  accountAliases, err := awslib.GetAccountAliases(awsConfig)
  if err == nil {
    log.Debug(logrus.Fields{"account": accountAliases, "region": region}, "craft-config startup.")
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
      log.WithFields(logrus.Fields{
        "key": k,
        "config-file": serverConfigFileName,
      }).Fatalf("Key \"%s\" not found in configuration \"%s\". No files updated.\n",k,serverConfigFileName)
    }
  }
}

func doArchiveAndPublish() {

  retries := 10
  waitTime := 5 * time.Second
  rcon, err := minecraft.NewRconWithRetry(serverIpArg, rconPortArg, rconPassword, retries, waitTime)
  server := serverIpArg + ":" + rconPortArg
  if err != nil {
    log.WithFields(logrus.Fields{
      "server": server, 
      "retries": retries, 
      "retryWait": waitTime,
    }).Error("RCON Connection failed. Can't archive")
    return
  }

  if continuousArchiveArg {
    continuousArchiveAndPublish(rcon, archiveDirectoryArg, bucketNameArg, userArg, awsConfig)
  } else {
    archiveAndPublish(rcon, archiveDirectoryArg, bucketNameArg, userArg, awsConfig)
  }
}

// TODO: Set up some asynchronous go routines:
// 1. Delay timer: every 5 mniutes or so, come along and do a backup if there are users (what we have now).
// 2. File Watcher: check to see if non-world files have been created and update those.
// 3. User Watcher: assuming the file watcher, can't proxy for this. Set a timeout for every 10 seconds and check for new users,
// update when one shows up.
//
// Finally  Put the whole thing in a go-routine that checks for a stop (see the watcher in ineteractive.)
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
      log.Infof("No users on server. Not updating the archive. Checking again in %s.", delayTime )
    }
    time.Sleep(delayTime)
  }
}

func archiveAndPublish(rcon *minecraft.Rcon, archiveDir, bucketName, user string, cfg *aws.Config) {
  archiveFields := logrus.Fields{"archiveDir": archiveDir,"bucket": bucketName, "user": user, }
  resp, err := minecraft.ArchiveAndPublish(rcon, archiveDir, bucketName, user, cfg)
  if err != nil {
    log.WithFields(archiveFields).WithError(err).Errorf("Error creating an archive and publishing to S3")
  } else {
    archiveFields["bucket"] = resp.BucketName
    archiveFields["archive"] = resp.StoredPath
    log.WithFields(archiveFields).Infof("Published archive to: %s:%s\n", resp.BucketName, resp.StoredPath)
  }
}
func configureLogs() {
  f := new(minecraft.TextFormatter)
  f.FullTimestamp = true
  log.Formatter = f
  log.Level = logrus.InfoLevel
}

func updateLogLevel() {
  l := logrus.InfoLevel
  if debug || verbose {
    l = logrus.DebugLevel
  }
  log.Level = l
  minecraft.SetLogLevel(l)
}

