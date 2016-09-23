package lib

import(
  "github.com/jdrivas/sl"
  "github.com/Sirupsen/logrus"
)

var log = sl.New()
func init() {
  configureLogs()
}

var debug = false
func setDebug(l logrus.Level) {
  switch l {
  case logrus.DebugLevel: debug = true
  default: debug = false
  }
}

func SetLogFormatter(f logrus.Formatter) { 
  log.SetFormatter(f)
}

func SetLogLevel(l logrus.Level) {
  log.SetLevel(l)
  setDebug(l)
}

func configureLogs() {
  formatter := new(sl.TextFormatter)
  formatter.FullTimestamp = true
  log.SetFormatter(formatter)
  log.SetLevel(logrus.InfoLevel)
}
