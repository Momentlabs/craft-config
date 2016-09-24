package main 

import(
  "fmt"
  "time"
  // "github.com/Sirupsen/logrus"

  // "mclib"
  "github.com/jdrivas/mclib"
)

func doArchiveAndPublish(server *mclib.Server) {
  retries := rconRetriesArg
  waitTime := time.Duration(rconDelayArg) * time.Second

  f := server.LogFields()
  if retries < 0  {
    count := 0
    for {
      server.NewRconWithRetry(10, waitTime)
      if server.HasRconConnection() {break}
      count++
      f["retryOuterLoop"] = count
      log.Debug(f,"Continuing to try to connect to rcon.")
    }
  } else {
      server.NewRconWithRetry(retries, waitTime)
  }

  if continuousArchiveArg {
    continuousArchiveAndPublish(server)
  } else {
    archiveAndPublish(server, mclib.ServerSnapshot)
  }
}

// TODO: Set up some asynchronous go routines:
// DONE 1. Delay timer: every 5 mniutes or so, come along and do a backup if there are users (what we have now).
// 2. File Watcher: check to see if non-world files have been created and update those.
// DONE 3. User Watcher: assuming the file watcher, can't proxy for this. Set a timeout for every 10 seconds 
// or so and check for new users, update when one shows up.
//
// Finally  Put the whole thing in a go-routine that checks for a stop (see the watcher in ineteractive.)
const backupDelay = 5 * time.Minute
const userCheckDelay = 30 * time.Second
const(
  newUser = iota
  backupTimeout
)
func continuousArchiveAndPublish(s *mclib.Server) {

  userFields := s.LogFields()
  userFields["users"] = 0
  userFields["delay"] = backupDelay.String()
  delayFields := s.LogFields()
  delayFields["delay"] = backupDelay.String()

  backupTimeoutCheck := time.Tick(backupDelay)
  newUserCheck := time.Tick(userCheckDelay)

  var err error
  lastUsers := 0
  currentUsers := 0
  wakeUpReason := newUser
  for {
    select {
    case <- newUserCheck:           
      wakeUpReason = newUser
    case <- backupTimeoutCheck:
      wakeUpReason = backupTimeout
    }

    // TODO: We may do a backup down here, synchronously.
    // This may cause the events above to backup (so far that's unlikely
    // as backups take about 1 sec. Don't know what,if anythnig happens if that
    // happens. But it's likely better than getting the asynchronous part wrong. 
    // We could don't wnat world saves (via the telling the server to say through rcon) 
    // to go off during another backup.

    // Don't do backups if there are no users.
    currentUsers, err = s.Rcon.NumberOfUsers();
    change := currentUsers != lastUsers
    if err != nil {
      log.Error(delayFields, "Can't get the number of users from the server. Will wait.", err)
      s.NewRcon() // TODO: Consider doing this with a retry.
    } else {
      userFields["users"] = currentUsers
      // If there are users, backup worlds and server every backuptimeout.
      // If we add or remove a user then catch that in a world backup.
      if currentUsers > 0 {
        if change && wakeUpReason == newUser {
          log.Info(userFields, "Archiving worlds.")
          archiveAndPublish(s, mclib.WorldSnapshot)
        } else if wakeUpReason == backupTimeout {
          log.Info(userFields, "Archiving worlds.")
          archiveAndPublish(s, mclib.WorldSnapshot)
          log.Info(userFields, "Archiving server.")
          archiveAndPublish(s, mclib.ServerSnapshot)
        } else {
          log.Info(userFields, "No changes. Not archiving")
        }
      } else {
        log.Info(userFields, "No users on server. Not archiving")
      }
    }
    lastUsers = currentUsers
  }
}

    // users, err := s.Rcon.NumberOfUsers()
    // if err != nil { 
    //   log.Error(nil, "Can't get the numbers of users from the server.", err)
    //   return
    // } 
    // if users > 0 {
    //   archiveAndPublish(s)
    // } else {
    //   log.Info(logrus.Fields{"retryDelay": delayTime.String(),}, "No users on server. Not updating the archive.")
    // }
    // time.Sleep(delayTime)


// Check for users, do the backup and report out.
func archiveAndPublish(s *mclib.Server, aType mclib.ArchiveType) {
  userFields := s.LogFields()
  userFields["serverDir"] = s.ServerDirectory
  userFields["bucket"] = s.ArchiveBucket
  userFields["archiveType"] = aType.String()

  var resp *mclib.PublishedArchiveResponse
  var err error
  switch aType {
  case mclib.ServerSnapshot: 
    log.Info(userFields,"Backup server.")
    resp, err = s.TakeServerSnapshot()
  case mclib.WorldSnapshot: 
    log.Info(userFields,"Backup World.")
    resp, err = s.TakeWorldSnapshot()
  default:
    resp = nil
    err = fmt.Errorf("Error archiving: Bad ArchiveType: %s", aType.String())
  }

  if err != nil {
    log.Error(userFields, "Error creating an archive and publishing to S3.", err)
  } else {
    userFields["archive"] = resp.StoredPath
    userFields["eTag"] =  *resp.PutObjectOutput.ETag
    log.Info(userFields, "Published archive.")
  }
}

