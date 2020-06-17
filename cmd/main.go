package main

import (
  "fmt"
  oracleclient "github.com/tcastelly/oracle-client-install/v1.0.1"
  "os"
)

func getRootPath() string {
  var rooPath string

  if len(os.Args) > 1 {
    rooPath = os.Args[1]
  } else {
    rooPath = ".oracle"
  }

  return rooPath
}

func main() {
  rootPath := getRootPath()

  if err := oracleclient.Uninstall(rootPath); err != nil {
    fmt.Println(err)
  } else if err = oracleclient.Install(rootPath); err != nil {
    fmt.Println(err)
  }
}
