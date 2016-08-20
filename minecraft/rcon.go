package minecraft

import(
  "fmt"
  // "io"
  // "net/http"
  "strconv"
  // "os"
  // "strings"
  // "time"
  // "archive/zip"
  // "path/filepath"
  "github.com/bearbin/mcgorcon"
  // "github.com/op/go-logging"
  )

type Rcon struct {
  Host string
  Port int
  Password string
  Client *mcgorcon.Client
}

// create a new connection.
func NewRcon(host string, port string, pw string) (rcon *Rcon, err error) {
  p, err := strconv.Atoi(port)
  if err == nil {
    rcon = &Rcon{
      Host: host, 
      Port: p,
      Password: pw,
    }
    client, err := mcgorcon.Dial(rcon.Host, rcon.Port, rcon.Password)
    log.Debugf("NewRcon: connected to server %s:%d", rcon.Host, rcon.Port)
    if err == nil {
      rcon.Client = &client
    }
  }
  return  rcon, err
}

func (rc *Rcon) Send(command string) (reply string, err error ) {
  if rc.Client == nil { return reply, fmt.Errorf("Rcon: Host connection empty.")}
  return rc.Client.SendCommand(command)
}

func (rc *Rcon) SaveOn() (reply string, err error){
  return rc.Send("save-on")
}

func (rc *Rcon) SaveOff() (reply string, err error) {
  return rc.Send("save-off")
}

func (rc *Rcon) SaveAll() (reply string, err error) {
  return rc.Send("save-all")
}


