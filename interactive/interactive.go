package interactive 

import (
  "gopkg.in/alecthomas/kingpin.v2"
  "github.com/bobappleyard/readline"
  "strings"
  "fmt"
  "io"
  "craft-config/minecraft"
  "github.com/aws/aws-sdk-go/aws"
  "github.com/op/go-logging"
  "github.com/fsnotify/fsnotify"

)

var (

  app *kingpin.Application
  testString []string

  exitCmd *kingpin.CmdClause
  quitCmd *kingpin.CmdClause
  verboseCmd *kingpin.CmdClause
  verbose bool

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

  log = logging.MustGetLogger("craft-config/minecraft")

  // watchDone bool
  watchDone chan bool
  watcher *fsnotify.Watcher
)

func init() {
  logging.SetLevel(logging.INFO, "craft-config/interactive")

  watchDone = make(chan bool)

  app = kingpin.New("", "Interactive mode.").Terminate(doTerminate)

  // state
  verboseCmd = app.Command("verbose", "toggle verbose mode.")
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
  archiveServerCmd.Arg("server-directory", "Relative location of server.").Default("server").StringVar(&serverDirectoryNameArg)
  archiveServerCmd.Arg("archive-file", "Name of archive file to create.").Default("server.zip").StringVar(&archiveFileNameArg)
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
      case exitCmd.FullCommand(): err = doQuit()
      case quitCmd.FullCommand(): err = doQuit()
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

func doArchiveServer() (error) {
  err := minecraft.ArchiveServer(serverDirectoryNameArg, archiveFileNameArg)
  return err
}

func doPublishArchive(awsConfig *aws.Config) (error) {
  resp, err := minecraft.PublishArchive(archiveFileNameArg, bucketNameArg, userNameArg, awsConfig)
  if err == nil {
    fmt.Printf("Published archive to: %s:%s\n", resp.BucketName, resp.StoredPath)
  }
  return err
}

func doWatchEventsStart() (err error) {
  if watcher != nil { return fmt.Errorf("Watcher already being used.") }

  watcher, err = fsnotify.NewWatcher()
  if err != nil { return fmt.Errorf("Couldn't create a notifycation watcher: %s", err) }

  go func() {
    log.Info("Starting file watch.")
    for {
      select {
      case event := <-watcher.Events:
        log.Infof("file watch event: %s", event)
        if event.Op & fsnotify.Write == fsnotify.Write {
          log.Infof("modified file: %s", event.Name)
        }
      case err := <-watcher.Errors:
        log.Infof("error: %s", err)
      case <-watchDone:
        log.Infof("Stopping file watch.")
        return
      } 
    }
  }()

  err = watcher.Add(".")
  return err
}

func doWatchEventsStop() (error) {
  if watcher == nil { return fmt.Errorf("No watcher to stop.")}
  log.Info("Shutting done the file watcher.")
  watchDone <- true
  log.Info("Closing the watcher.")
  watcher.Close()
  watcher = nil
  fmt.Printf("Stopping file watch.\n")
  return nil
}

// Interactive Mode support functions.
func toggleVerbose() bool {
  verbose = !verbose
  return verbose
}

func doVerbose() (error) {
  if toggleVerbose() {
    logging.SetLevel(logging.DEBUG, "craft-config/minecraft")
    fmt.Println("Verbose is on.")
  } else {
    logging.SetLevel(logging.INFO, "craft-config/minecraft")
    fmt.Println("Verbose is off.")
  }
  return nil
}

func doQuit() (error) {
  return io.EOF
}

func doTerminate(i int) {}

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
  xICommand := func(line string) (err error) {return doICommand(line, awsConfig)}
  prompt := "> "
  err := promptLoop(prompt, xICommand)
  if err != nil {fmt.Printf("Error - %s.\n", err)}
}




