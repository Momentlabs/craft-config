package minecraft
  
import (
  "os"
  // "log"
)

func checkFatalError(e error) {
  if e != nil {
    log.Critical(e)
    os.Exit(-1)
  }
}
