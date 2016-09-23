package lib

import(
  "fmt"
  "github.com/mgutz/ansi"
)

var (
  NullColor = fmt.Sprintf("%s", "\x00\x00\x00\x00\x00\x00\x00")
  DefaultColor = fmt.Sprintf("%s%s", "\x00\x00", ansi.ColorCode("default"))
  DefaultShortColor = fmt.Sprintf("%s", ansi.ColorCode("default"))

  EmphBlueColor = fmt.Sprintf(ansi.ColorCode("blue+b"))
  EmphRedColor = fmt.Sprintf(ansi.ColorCode("red+b"))
  EmphColor = EmphBlueColor

  TitleColor = fmt.Sprintf(ansi.ColorCode("default+b"))
  TitleEmph = EmphBlueColor
  InfoColor = EmphBlueColor
  SuccessColor = fmt.Sprintf(ansi.ColorCode("green+b"))
  WarnColor = fmt.Sprintf(ansi.ColorCode("yellow+b"))
  FailColor = EmphRedColor
  ResetColor = fmt.Sprintf(ansi.ColorCode("reset"))
)