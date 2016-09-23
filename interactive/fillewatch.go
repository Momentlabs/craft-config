package interactive

import(
  "path/filepath"
  "fmt"
  "os"
  "github.com/fsnotify/fsnotify"
  "github.com/Sirupsen/logrus"
)

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