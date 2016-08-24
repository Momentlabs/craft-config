package minecraft

import(
  "github.com/Sirupsen/logrus"
)

var(
  log = logrus.New()
)

func init() {
  // log.Formatter = new(logrus.JSONFormatter)
  formatter := new(TextFormatter)
  formatter.FullTimestamp = true
  log.Formatter = formatter
  log.Level = logrus.InfoLevel
}

func SetLogLevel(l logrus.Level) {
  log.Level= l
}