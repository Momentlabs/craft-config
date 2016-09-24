package interactive 


import (
  "fmt"
  "io"
  "strings"
  "strconv"
  "craft-config/version"
  "craft-config/lib"
  "github.com/alecthomas/kingpin"
  "github.com/aws/aws-sdk-go/aws/session"
  "github.com/fsnotify/fsnotify"
  "github.com/jdrivas/sl"
  "github.com/Sirupsen/logrus"

  // "mclib"
  "github.com/jdrivas/mclib"
)

const(
  defaultServerIp = "127.0.0.1"
  defaultPrivateIp = "127.0.0.1"
  defaultServerPort = mclib.Port(25565)
  defaultRconPort = mclib.Port(25575)
  defaultRconAddr = defaultServerIp + ":25575"
  defaultUser = "testuser"
  defaeultServerName = "craft-config-test-server"
  defaultArchiveBucket = "craft-config-test"
  defaultArchiveFile = "server.zip"

  defaultLogFormat = textLog

  NoArchiveTypeArg = "NoTypeArg"
)

var (

  app *kingpin.Application
  testString []string

  exitCmd *kingpin.CmdClause
  quitCmd *kingpin.CmdClause
  verboseCmd *kingpin.CmdClause
  versionCmd *kingpin.CmdClause
  logFormatCmd *kingpin.CmdClause
  verbose bool
  logFormatArg = defaultLogFormat
  debugCmd *kingpin.CmdClause
  debug bool

  queryCmd *kingpin.CmdClause
  queryCommandArg []string

  rcon *mclib.Rcon
  rconCmd *kingpin.CmdClause
  serverIpArg string
  rconAddrArg string
  rconPortArg string
  rconPasswordArg string
  noRcon bool

  // some variables that maintain state between command invoations
  // This requires login down in DoICommand()
  currentServerIp = defaultServerIp
  currentRconPort = defaultRconPort

  // Read a configuration file in the current config
  currentServerConfigFileNameArg string
  currentServerConfig *mclib.ServerConfig

  readServerConfigFileCmd *kingpin.CmdClause

  // Print the current configuration out.
  printServerConfigCmd *kingpin.CmdClause

  // Write the current configuraiton out.
  newServerConfigFileNameArg string
  writeServerConfigCmd *kingpin.CmdClause

  // Set a key value, key must already be present.
  setServerConfigValueCmd *kingpin.CmdClause
  currentKeyArg string
  currentValueArg string

  // Archive state
  archiveCmd *kingpin.CmdClause
  archiveServerCmd *kingpin.CmdClause
  archivePublishCmd *kingpin.CmdClause
  archiveGetCmd *kingpin.CmdClause
  archiveListCmd *kingpin.CmdClause

  archiveURIArg string
  archiveTypeArg string
  archiveFileNameArg string
  archiveFilesArg []string = make([]string, 0)
  serverDirectoryNameArg string
  bucketNameArg string
  userNameArg string
  serverNameArg string

  // Watch file-system.
  watchCmd *kingpin.CmdClause
  watchEventsCmd *kingpin.CmdClause
  watchEventsStartCmd *kingpin.CmdClause
  watchEventsStopCmd *kingpin.CmdClause

  // event Watcher control.
  watchDone chan bool
  watcher *fsnotify.Watcher

  log = sl.New()

)


func init() {
  watchDone = make(chan bool)

  app = kingpin.New("", "Interactive mode.").Terminate(func(int){})

  // state
  rconCmd = app.Command("rcon", "toggle rcon use.")
  versionCmd = app.Command("version", "Print out verison.")
  verboseCmd = app.Command("verbose", "toggle verbose mode.")
  debugCmd = app.Command("debug", "toggle the debug reporting.")
  exitCmd = app.Command("exit", "exit the program. <ctrl-D> works too.")
  quitCmd = app.Command("quit", "exit the program.")
  logFormatCmd = app.Command("log", "set the log format")
  logFormatCmd.Arg("format", "What format should we use").Default(defaultLogFormat).EnumVar(&logFormatArg, jsonLog, textLog)

  // Query a server.
  queryCmd = app.Command("query", "Use the rcon conneciton to query a running mc server.")
  queryCmd.Arg("rcon-address", "IP or DNS address for the rcon port of a server: minecraft:25575 or 172.31.55.58:25575").Default(defaultRconAddr).Action(setDefault).StringVar(&rconAddrArg)
  queryCmd.Flag("server-ip", "IP address or DNS name of the server.").Default(defaultServerIp).Action(setDefault).StringVar(&serverIpArg)
  queryCmd.Flag("rcon-port", "Port the server is listening for RCON connection.").Default("25575").StringVar(&rconPortArg)
  queryCmd.Flag("rcon-pw", "Password for the RCON connection.").Default("testing").StringVar(&rconPasswordArg)

  // Read and manipulate a configuration file.
  readServerConfigFileCmd = app.Command("read-config", "read a server config file in.")
  readServerConfigFileCmd.Arg("file-name", "The file to read the configuration file from.").Required().StringVar(&currentServerConfigFileNameArg)

  printServerConfigCmd = app.Command("print-config", "print the server config file.")

  writeServerConfigCmd = app.Command("write-config", "write the server config file.")
  writeServerConfigCmd.Arg("file-name", "The file to write the confiugration file to.").Required().StringVar(&newServerConfigFileNameArg)

  setServerConfigValueCmd = app.Command("set-config-value", "set a configuration value - key must already be present.")
  setServerConfigValueCmd.Arg("key", "Key for the setting - must be already presetn int he configuration").Required().StringVar(&currentKeyArg)
  setServerConfigValueCmd.Arg("value", "Value for the setting.").Required().StringVar(&currentValueArg)

  // Archive
  archiveCmd := app.Command("archive", "Context for managing archives.")

  archiveServerCmd = archiveCmd.Command("server", "Archive a server into a zip file.")
  archiveServerCmd.Arg("type", "Server or World snapshot.").Required().StringVar(&archiveTypeArg)
  archiveServerCmd.Arg("user", "Username for the server for archibing").Required().StringVar(&userNameArg)
  archiveServerCmd.Arg("server", "Servername for the server for archibing").Required().StringVar(&serverNameArg)
  archiveServerCmd.Arg("archive-files", "list of files to archive.").StringsVar(&archiveFilesArg)
  archiveServerCmd.Flag("bucket", "Name of S3 bucket to publish archive to.").Default(defaultArchiveBucket).StringVar(&bucketNameArg)
  archiveServerCmd.Flag("archive-file-name", "Name of archive (zip) file to create.").Default(defaultArchiveFile).StringVar(&archiveFileNameArg)
  archiveServerCmd.Flag("server-dir", "Relative location of server.").Default(".").StringVar(&serverDirectoryNameArg)
  archiveServerCmd.Flag("server-ip", "Server IP or dns. Used to get an RCON connection.").Default(defaultServerIp).StringVar(&serverIpArg)
  archiveServerCmd.Flag("rcon-port", "Port on the server where RCON is listening.").Default("25575").StringVar(&rconPortArg)
  archiveServerCmd.Flag("rcon-pw", "Password for rcon connection.").Default("testing").StringVar(&rconPasswordArg)
  archiveServerCmd.Flag("no-rcon","Don't try to connect to an RCON server for archiving. UNSAFE.").BoolVar(&noRcon)

  archivePublishCmd = archiveCmd.Command("publish", "Publish an archive to S3.")
  archivePublishCmd.Arg("user", "User of archive.").Required().StringVar(&userNameArg)
  archivePublishCmd.Arg("archive-file", "Name of archive file to pubilsh.").Default(defaultArchiveFile).StringVar(&archiveFileNameArg)
  archivePublishCmd.Arg("bucket", "Name of S3 bucket to publish archive to.").Default(defaultArchiveBucket).StringVar(&bucketNameArg)

  archiveGetCmd = archiveCmd.Command("get", "Retreive an archive from S3.")
  archiveGetCmd.Arg("uri", "Fullly qualified URI for the archive.").Required().StringVar(&archiveURIArg)
  // archiveListCmd.Arg("bucket", "Only list archives of this type.").Default(defaultArchiveBucket).StringVar(&bucketNameArg)

  archiveListCmd = archiveCmd.Command("list", "List the archives in the bucket.")
  archiveListCmd.Arg("user", "User name for the archives.").Required().StringVar(&userNameArg)
  archiveListCmd.Arg("type", "Only list archives of this type.").Default(NoArchiveTypeArg).StringVar(&archiveTypeArg)
  archiveListCmd.Arg("bucket", "Only list archives of this type.").Default(defaultArchiveBucket).StringVar(&bucketNameArg)

  // Watch
  watchCmd = app.Command("watch", "Watch the file system.")
  watchEventsCmd = watchCmd.Command("events", "Print out events.")
  watchEventsStartCmd = watchEventsCmd.Command("start", "Start watching events.")
  watchEventsStopCmd = watchEventsCmd.Command("stop", "Stop watching events.")

  configureLogs()
}


func doICommand(line string, sess *session.Session) (err error) {

  // Variables keep there values between parsings. This means that
  // slices of strings just grow. We reset them here.
  archiveFilesArg = []string{}

  // Prepare a line for parsing
  line = strings.TrimRight(line, "\n")
  fields := []string{}
  fields = append(fields, strings.Fields(line)...)
  if len(fields) <= 0 {
    return nil
  }

  command, err := app.Parse(fields)
  if err != nil {
    fmt.Printf("Command error: %s.\nType help for a list of commands.\n", err)
    return nil
  } else {
    // TODO: probably better served with map. Functions can take a vararg of interface{}.
    switch command {
      case verboseCmd.FullCommand(): err = doVerbose()
      case versionCmd.FullCommand(): err = doVersion()
      case debugCmd.FullCommand(): err = doDebug()
      case logFormatCmd.FullCommand(): err = doLogFormat()
      case exitCmd.FullCommand(): err = doQuit()
      case quitCmd.FullCommand(): err = doQuit()
      case rconCmd.FullCommand(): err = doRcon()
      case queryCmd.FullCommand(): err = doQuery()
      case readServerConfigFileCmd.FullCommand(): err = doReadServerConfigFile()
      case printServerConfigCmd.FullCommand(): err = doPrintServerConfig()
      case writeServerConfigCmd.FullCommand(): err = doWriteServerConfig()
      case setServerConfigValueCmd.FullCommand(): err = doSetServerConfigValue()
      case archiveServerCmd.FullCommand(): err = doArchiveServer(sess)
      case archivePublishCmd.FullCommand(): err = doPublishArchive(sess)
      case archiveGetCmd.FullCommand(): err = doGetArchive(sess)
      case archiveListCmd.FullCommand(): err = doListArchive(sess)
      case watchEventsStartCmd.FullCommand(): err = doWatchEventsStart()
      case watchEventsStopCmd.FullCommand(): err = doWatchEventsStop()
    }
  }
  return err
}

func doQuery() (error) {
  return lib.RconLoop(currentServerIp, currentRconPort, rconPasswordArg)
}

// TODO: This variables for currentServerIP etc. are getting a little crufty.
// this desparately needs some refactoring.
func setDefault(pc *kingpin.ParseContext) (error) {

  for _, pe := range pc.Elements {
    c := pe.Clause
    switch c.(type) {
      // case *kingpin.CmdClause : fmt.Printf("CmdClause: %s\n", (c.(*kingpin.CmdClause)).Model().Name)
    case *kingpin.ArgClause : {
      ac := c.(*kingpin.ArgClause)
      if ac.Model().Name == "rcon-address" {
        rconaddr := strings.Split(*pe.Value, ":")
        ip := rconaddr[0]
        port := defaultRconPort
        if len(rconaddr) > 1 {
          p, err := strconv.Atoi(rconaddr[1])
          if err != nil { p = 0 }
          port = mclib.Port(p)
        }
        currentServerIp = ip
        currentRconPort = port
      }
    }
    case *kingpin.FlagClause : 
      fc := c.(*kingpin.FlagClause)
      if fc.Model().Name == "server-ip" {
        currentServerIp = *pe.Value
      }
    }
  }

  return nil
}

func toggleNoRcon() bool {
  noRcon = !noRcon
  return noRcon
}

func toggleVerbose() bool {
  verbose = !verbose
  return verbose
}

func toggleDebug() bool {
  debug = !debug
  return debug
}

func doRcon() (error) {
  if toggleNoRcon() {
    fmt.Println("Rcon is turned off.")
  } else {
    fmt.Println("Rcon is turned on.")
  }
  return nil
}

func doVersion() (error) {
  fmt.Println(version.Version)
  return nil
}

func doVerbose() (error) {
  if toggleVerbose() {
    fmt.Println("Verbose is on.")
  } else {
    fmt.Println("Verbose is off.")
  }
  updateLogLevel()
  return nil
}

func doDebug() (error) {
  if toggleDebug() {
    fmt.Println("Debug is on.")
  } else {
    fmt.Println("Debug is off.")    
  }
  updateLogLevel()
  return nil
}

func doLogFormat() (error) {
  setFormatter()
  fmt.Printf("Log format is now: %s.\n", logFormatArg)
  return nil
}

func configureLogs() {
  setFormatter()
  updateLogLevel()
}

const (
  jsonLog = "json"
  textLog = "text"
  cliLog = "cli"
)

func setFormatter() {
  var f logrus.Formatter
  switch logFormatArg {
  case jsonLog: f = new(logrus.JSONFormatter)
  case cliLog, textLog:
    s := new(sl.TextFormatter)
    s.FullTimestamp = true
    f = logrus.Formatter(s)
  }
  log.SetFormatter(f)
  mclib.SetLogFormatter(f)
  lib.SetLogFormatter(f)
}

func updateLogLevel() {
  l := logrus.InfoLevel
  if debug || verbose {
    fmt.Printf("Setting debug LogLevel.\n")
    l = logrus.DebugLevel
  }
  log.SetLevel(l)
  mclib.SetLogLevel(l)
  lib.SetLogLevel(l)
}

func doQuit() (error) {
  return io.EOF
}

// This gets called from the main program, presumably from the 'interactive' command on main's command line.
func DoInteractive(sess *session.Session) {
  prompt := "craft-config > "
  err := lib.PromptLoop(prompt, func(line string) (err error) {
    return doICommand(line, sess)
  })
  if err != nil {fmt.Printf("Error - %s.\n", err)}
}




