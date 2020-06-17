package config

import (
  "errors"
  "runtime"
)

const BASE_DOWNLOAD = "https://github.com/shenron/docker-node/raw/master/oracle_client/conf/oracle/"

type Config struct {
  InstantclientBasic string

  InstantclientSdk string
}

func NewLinuxConfig() (Config, error) {
  os := runtime.GOOS

  var instantclientBasic string

  var instantclientSdk string

  var e error = nil

  switch os {
  case "darwin":
    instantclientBasic = "instantclient-basiclite-macos.x64-18.1.0.0.0.zip"
    instantclientSdk = "instantclient-sdk-macos.x64-18.1.0.0.0-2.zip"
  case "linux":
    instantclientBasic = "instantclient-basiclite-linux.x64-19.3.0.0.0dbru.zip"
    instantclientSdk = "instantclient-sdk-linux.x64-19.3.0.0.0dbru.zip"
  default:
    e = errors.New("Unsupported OS")
  }

  return Config{
    BASE_DOWNLOAD + instantclientBasic,

    BASE_DOWNLOAD + instantclientSdk,
  }, e
}
