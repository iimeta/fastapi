@echo off
cd ../
go env -w GO111MODULE=on

SET GOOS=linux
SET GOARCH=amd64

echo build start
go build -ldflags "-X main.Version=v0.1.0 -X main.Build=2024" -o ./bin/
echo build finish.