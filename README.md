# Download and install Oracle Client Driver


`instantclient_basic` and `instantclient_sdk` will be installed in the dir passed by arguments.
**Only Gnu Linux and OSX are supported OS**.

> go run cmd/main.go .oracle


Most of languages (Golang / NodeJs) can connect to the Oracle Client driver by setting `LD_LIBRARY_PATH`

> LD_LIBRARY_PATH=${pwd}/.oracle/instantclient
