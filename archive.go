package main 

import(

  "time"
  "github.com/Sirupsen/logrus"

  "mclib"
  // "github.com/jdrivas/mclib"
)

func doArchiveAndPublish(server *mclib.Server) {
  retries := rconRetriesArg
  waitTime := time.Duration(rconDelayArg) * time.Second

  if retries < 0  {
    count := 0
    for {
      server.NewRconWithRetry(10, waitTime)
      if server.HasRconConnection() {break}
      count++
      log.Debug(logrus.Fields{"retryOuterLoop": count,},"Continuing to try to connect to rcon.")
    }
  } else {
      server.NewRconWithRetry(retries, waitTime)
  }

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

  resp, err := s.TakeServerSnapshot()

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