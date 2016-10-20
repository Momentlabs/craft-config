package main 

import(
  "fmt"
  "time"
  "craft-config/version"
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

// TODO: Probably time to make these both command line paramaters.
const backupDelay = 5 * time.Minute
const userCheckDelay = 30 * time.Second
const(
  newUser = iota
  backupTimeout
)
func continuousArchiveAndPublish(s *mclib.Server) {

  f := s.LogFields()
  f["serverBackupTick"] = backupDelay.String()
  f["userCheckTick"] = userCheckDelay.String()
  f["controllerVersion"] = version.Version.String()
  f["operation"] = "SnapshotCheck"


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
      f["users"] = "<unknown>"
      log.Error(f, "Can't get the number of users from the server. Will wait.", err)
      s.NewRcon() // TODO: Consider doing this with a retry.
    } else {
      f["users"] = currentUsers
      // If there are users, backup worlds and server every backuptimeout.
      // If we add or remove a user then catch that in a world backup.
      // TODO: FOR THE MOMENT IT SEEMS CLEAR THAT WE SHOULD RESTART FROM
      // SERVER BACKUPs. Which means I need them to happen more often.
      if currentUsers > 0 {
        f["operation"] = "SnapshotCheck"
        if change && wakeUpReason == newUser {
          f["operation"] = "Snapshot"

          f["snapshotType"] = mclib.WorldSnapshot
          log.Info(f, "Taking snapshot.")
          archiveAndPublish(s, mclib.WorldSnapshot)

          f["snapshotType"] = mclib.Servernapshot
          log.Info(f, "Taking snapshot.")
          archiveAndPublish(s, mclib.ServerSnapshot)
        } else if wakeUpReason == backupTimeout {
          f["operation"] = "Snapshot"

          f["snapshotType"] = mclib.WorldSnapshot
          log.Info(f, "Taking snapshot.")
          archiveAndPublish(s, mclib.WorldSnapshot)

          f["snapshotType"] = mclib.Servernapshot
          log.Info(f, "Taking snapshot.")
          archiveAndPublish(s, mclib.ServerSnapshot)
        } else {
          f["snapshotType"] = "<none>"
          log.Info(f, "No changes. Not archiving")
        }
      } else {
        f["snapshotType"] = "<none>"
        log.Info(f, "No users on server. Not archiving")
      }
    }
    lastUsers = currentUsers
  }
}

// Check for users, do the backup and report out.
func archiveAndPublish(s *mclib.Server, aType mclib.ArchiveType) {
  f := s.LogFields()
  f["serverDir"] = s.ServerDirectory
  f["bucket"] = s.ArchiveBucket
  f["snapshotType"] = aType.String()
  f["operation"] = "Snapshot"

  var resp *mclib.PublishedArchiveResponse
  var err error
  switch aType {
  case mclib.ServerSnapshot: 
    resp, err = s.TakeServerSnapshot()
  case mclib.WorldSnapshot: 
    resp, err = s.TakeWorldSnapshot()
  default:
    resp = nil
    err = fmt.Errorf("Error archiving: Bad ArchiveType: %s", aType.String())
  }

  if err != nil {
    log.Error(f, "Error creating an archive and publishing to S3.", err)
  } else {
    f["uri"] = resp.URI()
    f["archive"] = resp.Key
    f["eTag"] =  *resp.PutObjectOutput.ETag
    f["result"] = "Success"
    log.Info(f, "Snapshot successful.")
  }
}

