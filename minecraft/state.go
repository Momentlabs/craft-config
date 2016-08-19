package minecraft

import(
  "fmt"
  "io"
  "os"
  "strings"
  "archive/zip"
  "path/filepath"
  "github.com/op/go-logging"
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
  if err != nil { return fmt.Errorf("ArchiveServer: can't change to directory %s: %s", directoryName, err) }

  fileNames := getServerFileNames()
  log.Debugf("ArchiveServer: will save %d entries to archive.\n", len(fileNames))
  for _, fileName := range fileNames {
    err = writeFileToZip(directoryName, fileName, archive)
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
    }

    if info.IsDir() {
      header.Name += "/"
    } else {
      header.Method = zip.Deflate // Is this necessary?
    }

    log.Debugf("Writing Header with Name: %s", header.Name)
    writer, err := archive.CreateHeader(header)
    if err != nil { return fmt.Errorf("Couldn't write header to archive: %s", err)}

    if !info.IsDir() {
        log.Debugf("Opening and copying file: %s", path)
        file, err := os.Open(path)
        if err != nil { fmt.Errorf("Couldn't open file %s: %s", path, err) }
        _, err = io.Copy(writer, file)
        if err != nil { return fmt.Errorf("io.copy failed: %s", path, err)}
    }

    return err
  })
  return err
}


// Replace .. and absolute paths in archives.
func sanitizedName(fileName string) string {
  fileName = filepath.ToSlash(fileName)
  fileName = strings.TrimLeft(fileName, "/.")
  return strings.Replace(fileName, "../", "", -1)
}

