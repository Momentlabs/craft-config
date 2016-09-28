package interactive

import(
  "fmt"
  "github.com/aws/aws-sdk-go/aws/session"
  "github.com/jdrivas/awslib"
)

func doAwsAccount(sess *session.Session) (error) {
  s, err := awslib.AccountDetailsString(sess.Config)
  if err != nil {
    fmt.Printf("Failed to get all the details: %s\n", err)
  }
  fmt.Printf("Details: %s\n", s)

  return nil
}
