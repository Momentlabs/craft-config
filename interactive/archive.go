package interactive

import(
  "fmt"
  "os"
  "sort"
  "strings"
  "time"
  "text/tabwriter"
  l "craft-config/lib"
  "github.com/aws/aws-sdk-go/aws/session"
  "mclib"
  // "github.com/jdrivas/mclib"
)

func doArchiveServer(sess *session.Session) (err error) {

  // Long winded here as a reminder about
  // where the variables are comming from and to make
  // it easier to make the change when it finally comes due.
  archiveType := mclib.ArchiveTypeFrom(archiveTypeArg)
  userName := userNameArg
  serverName := serverNameArg
  serverDirectory := serverDirectoryNameArg
  files := archiveFilesArg
  bucketName := bucketNameArg
  archiveFilename := archiveFileNameArg
  serverIp := serverIpArg
  rconPort := rconPortArg
  rp, err := mclib.NewPort(rconPort)
  if err != nil { rp = mclib.Port(0)}
  rconPw := rconPasswordArg

  s := &mclib.Server{
    User: userName,
    Name: serverName,
    PublicServerIp: serverIp,
    PrivateServerIp: defaultPrivateIp,
    ServerPort: defaultServerPort,
    RconPort:  rp,
    RconPassword: rconPw,
    ArchiveBucket: bucketName,
    ServerDirectory: serverDirectory,
    AWSSession: sess,
  }

  if archiveType == mclib.MiscSnapshot {
    if len(files) < 1 {
      return fmt.Errorf("Need at least one file for a  MiscSnapshot archive.")
    }
  } else {
    if len(files) > 0 {
      return fmt.Errorf("Cannot specify files for a %s archive", archiveType.String())
    }
  }
  

  w := tabwriter.NewWriter(os.Stdout, 4, 8, 3, ' ', 0)
  fmt.Printf("%s: Archiving Server.%s\n", l.TitleColor, l.ResetColor)
  fmt.Fprintf(w, "%sUser\tServer\tType\tServerDir\tBucket\tArchiveFile\tIP\tRconPort%s\n", l.TitleColor, l.ResetColor)
  fmt.Fprintf(w, "%s%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s%s\n", l.NullColor, 
    userName, serverName, archiveType, serverDirectory, 
    bucketName, archiveFilename, serverIp, rconPort, l.ResetColor)
  w.Flush()
  if len(files) > 0 {
    fmt.Printf("Files: %s\n", strings.Join(files, ","))
  }


  var resp *mclib.PublishedArchiveResponse
  switch archiveType {
  case mclib.ServerSnapshot:
    resp, err = s.TakeServerSnapshot()
  case mclib.WorldSnapshot: 
    resp, err = s.TakeWorldSnapshot()
  case mclib.MiscSnapshot:
    resp, err = s.TakeSnapshotWithFiles(files)
  default: 
    return fmt.Errorf("Error with incorrect archive type: %s", archiveType.String())
  }

  if err == nil {
    version := "----"
    if r := resp.PutObjectOutput.VersionId; r != nil { version = *r }
    etag := "----"
    if e := resp.PutObjectOutput.ETag; e != nil { etag = *e }
    w := tabwriter.NewWriter(os.Stdout, 4, 8, 3, ' ', 0)
    fmt.Printf("%s%sArchive Response.%s\n", l.TitleColor, time.Now().Local().Format(time.RFC1123), l.ResetColor)
    fmt.Fprintf(w, "%sUser\tBucket\tArchiveFile\tVersion\tEtag%s\n", l.TitleColor, l.ResetColor)
    fmt.Fprintf(w, "%s%s\t%s\t%s\t%s\t%s%s\n", l.NullColor,
      resp.UserName, resp.BucketName, resp.ArchiveFilename, version, etag, l.ResetColor)
    w.Flush()
    fmt.Printf("Path: %s\n", resp.StoredPath)
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

func doGetArchive(sess *session.Session) (error) {
  fmt.Printf("Not implemented yet.")
  return nil
}

func doListArchive(sess *session.Session) (err error) {
  userName := userNameArg
  bucketName := bucketNameArg

  var al []mclib.Archive
  if archiveTypeArg == NoArchiveTypeArg {
    am, err := mclib.GetArchives(userName, bucketName, sess )
    if err != nil { return err }
    al = am[userName]
  } else {
    t := mclib.ArchiveTypeFrom(archiveTypeArg)
    al, err = mclib.GetArchivesFor(t, userName, bucketName, sess)
  }
  sort.Sort(mclib.ByLastMod(al))

  w := tabwriter.NewWriter(os.Stdout, 4, 8, 3, ' ', 0)
  fmt.Printf("%s%s: %d Archives.%s\n", l.TitleColor, time.Now().Local().Format(time.RFC1123), len(al), l.ResetColor)
  fmt.Fprintf(w, "%sUser\tServer\tType\tBucket\tLastMod\tS3Key%s\n", l.TitleColor, l.ResetColor)
  for _, a := range al {
    fmt.Fprintf(w, "%s%s\t%s\t%s\t%s\t%s\t%s%s\n", l.NullColor, 
      a.UserName, a.ServerName, a.Type.String(), a.Bucket, a.LastMod(), a.S3Key(), l.ResetColor)
  }
  w.Flush()
  return nil
}
