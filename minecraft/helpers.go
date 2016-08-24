package minecraft
  
import (
  "os"
  // "log"
)

func checkFatalError(e error) {
  if e != nil {
    log.Fatal(e)
    os.Exit(-1)
  }
}
