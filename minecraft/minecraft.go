package minecraft

import (
  "github.com/jdrivas/sl"
  "github.com/Sirupsen/logrus"
)

var(
  // log = logrus.New()
  log = sl.New(logrus.New())
)

func init() {
  defaultConfigureLogs()
}

func SetLogLevel(l logrus.Level) {
  log.Logger.Level= l
}

func SetLogFormatter(f logrus.Formatter) {
  log.Logger.Formatter = f
}

func defaultConfigureLogs() {
  // log.Formatter = new(logrus.JSONFormatter)
  formatter := new(TextFormatter)
  formatter.FullTimestamp = true
  log.SetFormatter(formatter)
  // log.Logger.Level = logrus.InfoLevel
  log.SetLevel(logrus.InfoLevel)
}
