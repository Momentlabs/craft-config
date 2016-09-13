package interactive 


import (
  "fmt"
  "io"
  "os"
  "regexp"
  "strings"
  "strconv"
  "craft-config/version"
  "path/filepath"
  "github.com/alecthomas/kingpin"
  // "github.com/aws/aws-sdk-go/aws"
  "github.com/aws/aws-sdk-go/aws/session"
  "github.com/chzyer/readline"
  "github.com/fsnotify/fsnotify"
  "github.com/mgutz/ansi"
  "github.com/jdrivas/sl"
  "github.com/Sirupsen/logrus"

  // "github.com/jdrivas/awslib"
  "github.com/jdrivas/mclib"
)

const(
  defaultServerIp = "127.0.0.1"
  defaultRconPort = "25575"
  defaultRconAddr = defaultServerIp + ":" + defaultRconPort
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
  logFormatArg string
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
  archiveFileNameArg string
  serverDirectoryNameArg string
  bucketNameArg string
  userNameArg string

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


var (
  nullColor = fmt.Sprintf("%s", "\x00\x00\x00\x00\x00\x00\x00")
  defaultColor = fmt.Sprintf("%s%s", "\x00\x00", ansi.ColorCode("default"))
  defaultShortColor = fmt.Sprintf("%s", ansi.ColorCode("default"))

  emphBlueColor = fmt.Sprintf(ansi.ColorCode("blue+b"))
  emphRedColor = fmt.Sprintf(ansi.ColorCode("red+b"))
  emphColor = emphBlueColor

  titleColor = fmt.Sprintf(ansi.ColorCode("default+b"))
  titleEmph = emphBlueColor
  infoColor = emphBlueColor
  successColor = fmt.Sprintf(ansi.ColorCode("green+b"))
  warnColor = fmt.Sprintf(ansi.ColorCode("yellow+b"))
  failColor = emphRedColor
  resetColor = fmt.Sprintf(ansi.ColorCode("reset"))
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
  logFormatCmd.Arg("format", "What format should we use").Default(textLog).EnumVar(&logFormatArg, jsonLog, textLog)

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
  archiveServerCmd.Flag("no-rcon","Don't try to connect to an RCON server for archiving. UNSAFE.").BoolVar(&noRcon)
  archiveServerCmd.Arg("server-directory", "Relative location of server.").Default("server").StringVar(&serverDirectoryNameArg)
  archiveServerCmd.Arg("archive-file", "Name of archive file to create.").Default("server.zip").StringVar(&archiveFileNameArg)
  archiveServerCmd.Arg("server-ip", "Server IP or dns. Used to get an RCON connection.").Default(defaultServerIp).StringVar(&serverIpArg)
  archiveServerCmd.Arg("rcon-port", "Port on the server where RCON is listening.").Default("25575").StringVar(&rconPortArg)
  archiveServerCmd.Arg("rcon-pw", "Password for rcon connection.").Default("testing").StringVar(&rconPasswordArg)

  archivePublishCmd = archiveCmd.Command("publish", "Publish and archive to S3.")
  archivePublishCmd.Arg("user", "User of archive.").Required().StringVar(&userNameArg)
  archivePublishCmd.Arg("archive-file", "Name of archive file to pubilsh.").Default("server.zip").StringVar(&archiveFileNameArg)
  archivePublishCmd.Arg("s3-bucket", "Name of S3 bucket to publish archive to.").Default("craft-config-test").StringVar(&bucketNameArg)

  // Watch
  watchCmd = app.Command("watch", "Watch the file system.")
  watchEventsCmd = watchCmd.Command("events", "Print out events.")
  watchEventsStartCmd = watchEventsCmd.Command("start", "Start watching events.")
  watchEventsStopCmd = watchEventsCmd.Command("stop", "Stop watching events.")

  configureLogs()
}


func doICommand(line string, sess *session.Session) (err error) {

  // This is due to a 'peculiarity' of kingpin: it collects strings as arguments across parses.
  testString = []string{}

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
      case archiveServerCmd.FullCommand(): err = doArchiveServer()
      case archivePublishCmd.FullCommand(): err = doPublishArchive(sess)
      case watchEventsStartCmd.FullCommand(): err = doWatchEventsStart()
      case watchEventsStopCmd.FullCommand(): err = doWatchEventsStop()
    }
  }
  return err
}

// Interactive Command processing

func doQuery() (error) {
  rcon, err := mclib.NewRcon(currentServerIp, currentRconPort, rconPasswordArg)    
  if err != nil {return err}

  prompt := fmt.Sprintf("%s%s:%s%s: ", infoColor, currentServerIp, currentRconPort, resetColor)
  err = promptLoop(prompt, func(line string) (error) {
    if strings.Compare(line, "quit") == 0 || strings.Compare(line, "exit") == 0 {return io.EOF}
    if strings.Compare(line, "stop") == 0 || strings.Compare(line, "end") == 0 {
      return fmt.Errorf("Can't shutdown the server from here")
    }

    resp, err := rcon.Send(line)
    if err != nil { return err }
    if debug { 
      rs := strconv.Quote(resp) 
      fmt.Printf("%s%s:%s [RAW]%s: %s\n", infoColor, currentServerIp, currentRconPort, resetColor, rs)
    }
    fmt.Printf("%s\n", formatRconResp(resp))
    return err
  })

  return err
}

// Takes the color coding out.
func formatRconResp(r string) (s string) {
  re := regexp.MustCompile("§.")
  s = re.ReplaceAllString(r, "", )
  return s
}

func doReadServerConfigFile() (error) {
  currentServerConfig = mclib.NewConfigFromFile(currentServerConfigFileNameArg)
  return nil
}

func doPrintServerConfig() (error) {
  currentServerConfig.List()
  return nil
}

func doWriteServerConfig() (error) {
  if verbose {
    fmt.Printf("Writing out file: \"%s\"", newServerConfigFileNameArg)
  }
  currentServerConfig.WriteToFile(newServerConfigFileNameArg)
  return nil
}

func doSetServerConfigValue() (error) {
  currentServerConfig.SetEntry(currentKeyArg, currentValueArg)
  return nil
}

func doArchiveServer() (err error) {

  // This panics?
  // connected := (rcon != nil) || rcon.HasConnection()
  connected := false
  if rcon != nil {
    connected = rcon.HasConnection()
  }

  if noRcon {
    log.Info(nil, "Archiving without stopping saves on the server (no RCON connection). This is unsafe.")
    err = mclib.CreateServerArchive(serverDirectoryNameArg, archiveFileNameArg)
  } else {
    if !connected {
      rcon, err = mclib.NewRcon(serverIpArg, rconPortArg, rconPasswordArg)
      if err != nil { return fmt.Errorf("Can't open rcon connection to server %s:%s %s", serverIpArg, rconPortArg, err) }
    }
    err = mclib.ArchiveServer(rcon, serverDirectoryNameArg, archiveFileNameArg)
  }
  return err
}

func doPublishArchive(sess *session.Session) (error) {
  resp, err := mclib.PublishArchive(archiveFileNameArg, bucketNameArg, userNameArg, sess)
  if err == nil {
    fmt.Printf("Published archive to: %s:%s\n", resp.BucketName, resp.StoredPath)
  }
  return err
}


// TODO: Either add a .craftignore file
// or at least don't look at .git.
func doWatchEventsStart() (err error) {
  if watcher != nil { return fmt.Errorf("Watcher already being used.") }

  watcher, err = fsnotify.NewWatcher()
  if err != nil { return fmt.Errorf("Couldn't create a notifycation watcher: %s", err) }

  go func() {
    log.Info(nil, "Starting file watch.")
    for {
      select {
      case event := <-watcher.Events:
        log.Info(logrus.Fields{"event": event}, "File Event")
        if event.Op & fsnotify.Create == fsnotify.Create { // If we add a dir, watch it.
          file, err := os.Open(event.Name)
          f := logrus.Fields{"file": event.Name}
          if err != nil {log.Error(f, "Can't open new file.", err)}
          fInfo, err := file.Stat()
          if err != nil {log.Error(f, "Can't state new file.", err)}
          if fInfo.IsDir() {
            log.Info(f, "Adding directory to watch.")
            watcher.Add(event.Name)
          }
        }
      case err := <-watcher.Errors:
        log.Error(nil, "File watch.", err)
      case <-watchDone:
        log.Info(nil, "Stopping file watch.")
        return
      } 
    }
  }()
  addWatchTree(".", watcher)
  return err
}

// add the directories starting at the base to a watcher.
func addWatchTree(baseDir string, w *fsnotify.Watcher) (err error) {

  f := logrus.Fields{ "watchDir": baseDir, "file": ""}
  log.Debug(f, "Adding files to directory.")
  err = filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) (error) {
    // fds["file"] = path
    // ctx.Debug("Adding a file.")
    f["file"] = path
    log.Debug(f, "Adding a file.")
    if err != nil { return err }
    if info.IsDir() {
      // log.Infof("Adding %s to watch list.", path)
      err = w.Add(path)
    }
    return err
  })
  return err
}

func doWatchEventsStop() (error) {
  if watcher == nil { return fmt.Errorf("No watcher to stop.")}
  log.Debug(nil, "Shutting done the file watcher.")
  watchDone <- true
  log.Debug(nil, "Closing the watcher.")
  watcher.Close()
  watcher = nil
  fmt.Printf("File watch stopped.\n")
  return nil
}

//
// Interactive Mode support functions.
//


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
          port = rconaddr[1]
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

// TODO: Clearly this should set a *logrus.Formatter in the switch
// and then set each of the loggers to that formater.
// The foramtters are of different types though, and 
// lorgus.Formatter is an interface so I can't create a 
// var out of it. I konw there is a way, I just 
// don't know what it is.
func setFormatter() {
  switch logFormatArg {
  case jsonLog:
    f := new(logrus.JSONFormatter)
    log.SetFormatter(f)
    mclib.SetLogFormatter(f)
  case textLog:
    f := new(sl.TextFormatter)
    f.FullTimestamp = true
    log.SetFormatter(f)
    mclib.SetLogFormatter(f)
  default:
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

// func updateLogSettings() {
//   logLevel := logrus.InfoLevel
//   if verbose || debug {
//     logLevel = logrus.DebugLevel
//   }
//   log.WithField("loglevel", logLevel).Info("Setting log level.")
//   log.Level = logLevel
//   mclib.SetLogLevel(logLevel)
//   // log.Formatter = new(logrus.JSONFormatter)
//   log.Formatter = new(sl.TextFormatter)
//   logrus.SetLevel(logrus.DebugLevel)
// }

func doQuit() (error) {
  return io.EOF
}

func promptLoop(prompt string, process func(string) (error)) (err error) {
  errStr := "Error - %s.\n"
  for moreCommands := true; moreCommands; {
    line, err := readline.Line(prompt)
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
func DoInteractive(sess *session.Session) {
  prompt := "> "
  err := promptLoop(prompt, func(line string) (err error) {
    return doICommand(line, sess)
  })
  if err != nil {fmt.Printf("Error - %s.\n", err)}
}




