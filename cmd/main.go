package main

import (
  "fmt"
  oracleclient "github.com/tcastelly/oracle-client-install"
  "log"
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
  } else {
    install(rootPath)
  }
}

/**
 * `Install` is a wrapper of `InstallWithCh`
 */
func install(rootPath string) {
  err := oracleclient.Install(rootPath)
  if err != nil {
    log.Fatalf("%v", err)
  }
}

