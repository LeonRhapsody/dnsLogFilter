#! /bin/bash
envArgs="-X 'main.Commit=$(git rev-parse --short HEAD)' -X 'main.GitLog=$(git log -1)' -X 'main.BuildTime=$(date +"%F %H:%M:%S")'"

#linux
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "$envArgs" -o logTransfer ;mv logTransfer ~/Downloads

#macos
go build -ldflags "$envArgs" -o logTransfer

./logTransfer version
