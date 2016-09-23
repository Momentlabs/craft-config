package lib

import(
  "fmt"
  "io"
  "regexp"
  "strconv"
  "strings"
  "github.com/chzyer/readline"

  "mclib"
  // "github.com/jdrivas/mclib"
)

// TODO: Put this into mclib.

// Blocks on reading commands and writing input to the stdin/out
func RconLoop(serverIp string, rconPort mclib.Port, rconPassword string) (error) {

  rcon, err := mclib.NewRcon(serverIp, rconPort.String(), rconPassword)    
  if err != nil {return err}

  prompt := fmt.Sprintf("%s%s:%s%s: ", EmphColor, serverIp, rconPort, ResetColor)
  err = PromptLoop(prompt, func(line string) (error) {
    if strings.Compare(line, "quit") == 0 || strings.Compare(line, "exit") == 0 {return io.EOF}
    if strings.Compare(line, "stop") == 0 || strings.Compare(line, "end") == 0 {
      return fmt.Errorf("Can't shutdown the server from here")
    }

    resp, err := rcon.Send(line)
    if err != nil { return err }
    if debug { 
      rs := strconv.Quote(resp) 
      fmt.Printf("%s%s:%s [RAW]%s: %s\n", EmphColor, serverIp, rconPort, ResetColor, rs)
    }
    fmt.Printf("%s\n", formatRconResp(resp))
    return err
  })

  return err

}

// Takes the color coding out.
func formatRconResp(r string) (s string) {
  re := regexp.MustCompile("§.")
  s = re.ReplaceAllString(r, "", )
  return s
}


func PromptLoop(prompt string, process func(string) (error)) (err error) {
  errStr := "Error - %s.\n"
  for moreCommands := true; moreCommands; {
    line, err := readline.Line(prompt)
    if err == io.EOF {
      moreCommands = false
    } else if err != nil {
      fmt.Printf(errStr, err)
    } else {
      readline.AddHistory(line)
      err = process(line)
      if err == io.EOF {
        moreCommands = false
      } else if err != nil {
        fmt.Printf(errStr, err)
      }
    }
  }
  return nil
}