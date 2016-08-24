package interactive 


import (
  "gopkg.in/alecthomas/kingpin.v2"
  "github.com/bobappleyard/readline"
  "strings"
  "fmt"
  "io"
  "os"
  "path/filepath"
  "craft-config/minecraft"
  "github.com/aws/aws-sdk-go/aws"
  "github.com/fsnotify/fsnotify"
  "github.com/jdrivas/sl"
  "github.com/Sirupsen/logrus"
)

var (

  app *kingpin.Application
  testString []string

  exitCmd *kingpin.CmdClause
  quitCmd *kingpin.CmdClause
  verboseCmd *kingpin.CmdClause
  verbose bool
  debugCmd *kingpin.CmdClause
  debug bool

  rcon *minecraft.Rcon
  rconCmd *kingpin.CmdClause
  serverIpArg string
  rconPortArg string
  rconPassword string
  noRcon bool

  // Read a configuration file in the current config
  currentServerConfigFileNameArg string
  currentServerConfig *minecraft.ServerConfig

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

  log = logrus.New()


  // event Watcher control.
  watchDone chan bool
  watcher *fsnotify.Watcher

)

func init() {


  watchDone = make(chan bool)

  app = kingpin.New("", "Interactive mode.").Terminate(func(int){})

  // state
  rconCmd = app.Command("rcon", "toggle rcon use.")
  verboseCmd = app.Command("verbose", "toggle verbose mode.")
  debugCmd = app.Command("debug", "toggle the debug reporting.")
  exitCmd = app.Command("exit", "exit the program. <ctrl-D> works too.")
  quitCmd = app.Command("quit", "exit the program.")

  // Read and manipulate a configuration file.
  readServerConfigFileCmd = app.Command("read-config", "read a server config file in.")
  readServerConfigFileCmd.Arg("file-name", "The file to read the configuration file from.").Required().StringVar(&currentServerConfigFileNameArg)

  printServerConfigCmd = app.Command("print-config", "print the server config file.")

  writeServerConfigCmd = app.Command("write-config", "write the server config file.")
  writeServerConfigCmd.Arg("file-name", "The file to write the confiugration file to.").Required().StringVar(&newServerConfigFileNameArg)

  setServerConfigValueCmd = app.Command("set-config-value", "set a configuration value - key must already be present.")
  setServerConfigValueCmd.Arg("key", "Key for the setting - must be already presetn int he configuration").Required().StringVar(&currentKeyArg)
  setServerConfigValueCmd.Arg("value", "Value for the setting.").Required().StringVar(&currentValueArg)

  archiveCmd := app.Command("archive", "Context for managing archives.")
  archiveServerCmd = archiveCmd.Command("server", "Archive a server into a zip file.")
  archiveServerCmd.Flag("no-rcon","Don't try to connect to an RCON server for archiving. UNSAFE.").BoolVar(&noRcon)
  archiveServerCmd.Arg("server-directory", "Relative location of server.").Default("server").StringVar(&serverDirectoryNameArg)
  archiveServerCmd.Arg("archive-file", "Name of archive file to create.").Default("server.zip").StringVar(&archiveFileNameArg)
  archiveServerCmd.Arg("server-ip", "Server IP or dns. Used to get an RCON connection.").Default("127.0.0.1").StringVar(&serverIpArg)
  archiveServerCmd.Arg("rcon-port", "Port on the server where RCON is listening.").Default("25575").StringVar(&rconPortArg)
  archiveServerCmd.Arg("rcon-ps", "Password for rcon connection.").Default("testing").StringVar(&rconPassword)
  archivePublishCmd = archiveCmd.Command("publish", "Publish and archive to S3.")
  archivePublishCmd.Arg("user", "User of archive.").Required().StringVar(&userNameArg)
  archivePublishCmd.Arg("archive-file", "Name of archive file to pubilsh.").Default("server.zip").StringVar(&archiveFileNameArg)
  archivePublishCmd.Arg("s3-bucket", "Name of S3 bucket to publish archive to.").Default("craft-config-test").StringVar(&bucketNameArg)

  watchCmd = app.Command("watch", "Watch the file system.")
  watchEventsCmd = watchCmd.Command("events", "Print out events.")
  watchEventsStartCmd = watchEventsCmd.Command("start", "Start watching events.")
  watchEventsStopCmd = watchEventsCmd.Command("stop", "Stop watching events.")

}


func doICommand(line string, awsConfig *aws.Config) (err error) {

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
    switch command {
      case verboseCmd.FullCommand(): err = doVerbose()
      case debugCmd.FullCommand(): err = doDebug()
      case exitCmd.FullCommand(): err = doQuit()
      case quitCmd.FullCommand(): err = doQuit()
      case rconCmd.FullCommand(): err = doRcon()
      case readServerConfigFileCmd.FullCommand(): err = doReadServerConfigFile()
      case printServerConfigCmd.FullCommand(): err = doPrintServerConfig()
      case writeServerConfigCmd.FullCommand(): err = doWriteServerConfig()
      case setServerConfigValueCmd.FullCommand(): err = doSetServerConfigValue()
      case archiveServerCmd.FullCommand(): err = doArchiveServer()
      case archivePublishCmd.FullCommand(): err = doPublishArchive(awsConfig)
      case watchEventsStartCmd.FullCommand(): err = doWatchEventsStart()
      case watchEventsStopCmd.FullCommand(): err = doWatchEventsStop()
    }
  }
  return err
}

// Interactive Command processing
func doReadServerConfigFile() (error) {
  currentServerConfig = minecraft.NewConfigFromFile(currentServerConfigFileNameArg)
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
    log.Info("Archiving without stopping saves on the server (no RCON connection). This is unsafe.")
    err = minecraft.CreateServerArchive(serverDirectoryNameArg, archiveFileNameArg)
  } else {
    if !connected {
      rcon, err = minecraft.NewRcon(serverIpArg, rconPortArg, rconPassword)
      if err != nil { return fmt.Errorf("Can't open rcon connection to server %s:%s %s", serverIpArg, rconPortArg, err) }
    }
    err = minecraft.ArchiveServer(rcon, serverDirectoryNameArg, archiveFileNameArg)
  }
  return err
}

func doPublishArchive(awsConfig *aws.Config) (error) {
  resp, err := minecraft.PublishArchive(archiveFileNameArg, bucketNameArg, userNameArg, awsConfig)
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
    log.Info("Starting file watch.")
    for {
      select {
      case event := <-watcher.Events:
        log.Infof("%s", event)
        if event.Op & fsnotify.Create == fsnotify.Create { // If we add a dir, watch it.
          file, err := os.Open(event.Name)
          if err != nil {log.Errorf("Can't open new file %s: %s", event.Name, err)}
          fInfo, err := file.Stat()
          if err != nil {log.Errorf("Can't state new file %s: %s", event.Name, err)}
          if fInfo.IsDir() {
            log.Infof("Adding director %s to watch.", event.Name)
            watcher.Add(event.Name)
          }
        }
      case err := <-watcher.Errors:
        log.Infof("error: %s", err)
      case <-watchDone:
        log.Infof("Stopping file watch.")
        return
      } 
    }
  }()
  addWatchTree(".", watcher)
  return err
}

// add the directories starting at the base to a watcher.
func addWatchTree(baseDir string, w *fsnotify.Watcher) (err error) {
  // f := log.Fields{
  //   "watchDir": baseDir,
  //   "file": "",    
  // }.Fields()
  // ctx := log.WithFields(f)
  // fds := f.Fields()

  // ctx.Debug("Adding files to directory.")
  f := logrus.Fields{ "watchDir": baseDir, "file": ""}
  log.WithFields(f).Debug("Adding files to directory.")
  err = filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) (error) {
    // fds["file"] = path
    // ctx.Debug("Adding a file.")
    f["file"] = path
    log.WithFields(f).Debug("Adding a file.")
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
  log.Debugf("Shutting done the file watcher.")
  watchDone <- true
  log.Debugf("Closing the watcher.")
  watcher.Close()
  watcher = nil
  fmt.Printf("File watch stopped.\n")
  return nil
}

// Interactive Mode support functions.
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

func doVerbose() (error) {
  if toggleVerbose() {
    fmt.Println("Verbose is on.")
  } else {
    fmt.Println("Verbose is off.")
  }
  updateLogSettings()
  return nil
}

func doDebug() (error) {
  if toggleDebug() {
    fmt.Println("Debug is on.")
  } else {
    fmt.Println("Debug is off.")    
  }
  updateLogSettings()
  return nil
}

func updateLogSettings() {
  logLevel := logrus.InfoLevel
  if verbose || debug {
    logLevel = logrus.DebugLevel
  }
  log.WithField("loglevel", logLevel).Info("Setting log level.")
  log.Level = logLevel
  minecraft.SetLogLevel(logLevel)
  // log.Formatter = new(logrus.JSONFormatter)
  log.Formatter = new(sl.TextFormatter)
  logrus.SetLevel(logrus.DebugLevel)
}

func doQuit() (error) {
  return io.EOF
}

func promptLoop(prompt string, process func(string) (error)) (err error) {
  errStr := "Error - %s.\n"
  for moreCommands := true; moreCommands; {
    line, err := readline.String(prompt)
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
func DoInteractive(awsConfig *aws.Config) {
  prompt := "> "
  err := promptLoop(prompt, func(line string) (err error) {
    return doICommand(line, awsConfig)
  })
  if err != nil {fmt.Printf("Error - %s.\n", err)}
}




