package minecraft
  
import (
  "log"
)

func checkFatalError(e error) {
  if e != nil {
    log.Fatal(e)
  }
}
