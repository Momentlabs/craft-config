package minecraft

import(
  "bytes"
  "fmt"
  "io"
  "net/http"
  "os"
  "strings"
  "time"
  "archive/zip"
  "path/filepath"
  "github.com/op/go-logging"
  "github.com/aws/aws-sdk-go/aws"
  "github.com/aws/aws-sdk-go/aws/session"
  "github.com/aws/aws-sdk-go/service/s3"
  // "github.com/rlmcpherson/s3gof3r"
  // "golang.org/x/exp/inotify"
)


var (
  log = logging.MustGetLogger("craft-config/minecraft")
)

func init() {
  // logging.SetLevel(logging.DEBUG, "craft-config/minecraft")
  logging.SetLevel(logging.INFO, "craft-config/minecraft")
}

// Make a zipfile of the server directory in directoryName.
func ArchiveServer(directoryName, zipfileName string) (err error) {

  log.Debugf("ArchiveServer: going to archive %s to %s\n", directoryName, zipfileName)
  zipFile, err := os.Create(zipfileName)
  if err != nil { return fmt.Errorf("ArchiveServer: can't open zipfile %s: %s", zipfileName, err) }
  defer zipFile.Close()
  archive := zip.NewWriter(zipFile)
  defer archive.Close()

  dir, err := os.Open(directoryName)
  if err != nil { return fmt.Errorf("ArchiveServer: can't open server directory %s: %s", directoryName, err) }
  dirInfo, err := dir.Stat()
  if err != nil { return fmt.Errorf("ArchiveServer: can't stat directory %s: %s", directoryName, err) }
  if !dirInfo.IsDir() { return fmt.Errorf("ArchiveServer: server directory %s is not a directory.") }

  currentDir, err := os.Getwd()
  if err != nil { return fmt.Errorf("ArchiveServer: can't get the current directory: %s", err) }
  defer os.Chdir(currentDir)

  err = dir.Chdir()
  if err != nil { return fmt.Errorf("ArchiveServer: can't change to server directory %s: %s", directoryName, err) }

  fileNames := getServerFileNames()
  log.Debugf("ArchiveServer: will save %d entries to archive.\n", len(fileNames))
  for _, fileName := range fileNames {
    err = writeFileToZip("", fileName, archive)
    // err = writeFileToZip(directoryName, fileName, archive)
    if err != nil {return fmt.Errorf("ArchiveServer: can't write file \"%s\" to archive: %s", fileName, err)}
  }
  return err
}

func getServerFileNames() []string {
  files := []string{
    "config",
    "logs",
    "mods",
    "world",
    "banned-ips.json",
    "banned-players.json",
    "server.properties",
    "usercache.json",
    "whitelist.json",
  }
  return files
}
  
func writeFileToZip(baseDir, fileName string, archive *zip.Writer) (err error) {

  err = filepath.Walk(fileName, func(path string, info os.FileInfo, err error) (error) {
    if err != nil { return err }

    header, err := zip.FileInfoHeader(info)
    if err != nil { return fmt.Errorf("Couldn't convert FileInfo into zip header: %s", err) }

    if baseDir != "" {
      header.Name = filepath.Join(baseDir, path)
    } else {
      header.Name = path
    }

    if info.IsDir() {
      header.Name += "/"
    } else {
      header.Method = zip.Deflate // Is this necessary?
    }

    log.Debugf("Writing Zip Header with Name: %s", header.Name)
    writer, err := archive.CreateHeader(header)
    if err != nil { return fmt.Errorf("Couldn't write header to archive: %s", err)}

    if !info.IsDir() {
        log.Debugf("Opening and copying file to archive: %s", path)
        file, err := os.Open(path)
        if err != nil { fmt.Errorf("Couldn't open file %s: %s", path, err) }
        _, err = io.Copy(writer, file)
        if err != nil { return fmt.Errorf("io.copy failed: %s", path, err)}
    }

    return err
  })
  return err
}

type PublishedArchiveResponse struct {
  ArchiveFilename string
  BucketName string
  StoredPath string
  UserName string
  PutObjectOutput *s3.PutObjectOutput
}
// Puts the archive in the provided bucket on S3 in a 'directory' for the user. Bucket  must already exist.
// Config must have keys and region.
// The structure in the bucket is:
//    bucket:/<username>/archives/<ansi-time-string>-<username>-archive
func PublishArchive(archiveFileName string, bucketName string, user string, config *aws.Config) (*PublishedArchiveResponse, error) {
  s3svc := s3.New(session.New(config))
  file, err := os.Open(archiveFileName)
  if err != nil {return nil, fmt.Errorf("PublishArchive: Couldn't open archive file: %s: %s", archiveFileName, err)}
  defer file.Close()

  fileInfo, err := file.Stat()
  if err != nil {return nil, fmt.Errorf("PublishArchive: Couldn't stat archive file: %s: %s", archiveFileName, err)}
  fileSize := fileInfo.Size()

  buffer := make([]byte, fileSize)
  fileType := http.DetectContentType(buffer)
  _, err = file.Read(buffer)
  if err != nil {return nil, fmt.Errorf("PublishArchive: Couldn't read archive file: %s: %s", archiveFileName, err)}
  fileBytes := bytes.NewReader(buffer)

  path := getArchiveName(user)
  log.Debugf("PublishArchive: writing %s with %d bytes, type: %s to %s:%s", archiveFileName, fileSize, fileType, bucketName, path)

  // TOTO: Lookinto this and in particular figure out how to use an iamrole for this.
  aclString := "public-read"

  params := &s3.PutObjectInput{
    Bucket: aws.String(bucketName),
    Key: aws.String(path),
    ACL: aws.String(aclString),
    Body: fileBytes,
    ContentLength: aws.Int64(fileSize),
    ContentType: aws.String(fileType),
  }
  resp, err := s3svc.PutObject(params)

  returnResp := &PublishedArchiveResponse{
    ArchiveFilename: archiveFileName,
    BucketName: bucketName,
    StoredPath: path,
    UserName: user,
    PutObjectOutput: resp,
  }

  return returnResp, err
}

// TODO: Need to obscure this if we're going to make it publicly readable.
func getArchiveName(user string) string {
  timeString := time.Now().Format(time.RFC3339)
  return user + "/archives/" + timeString + "-" + user + "-archive"
}

// Replace .. and absolute paths in archives.
func sanitizedName(fileName string) string {
  fileName = filepath.ToSlash(fileName)
  fileName = strings.TrimLeft(fileName, "/.")
  return strings.Replace(fileName, "../", "", -1)
}

