@echo off
cd ../
go env -w GO111MODULE=on

REM 根据当前日期获取，年月日串
SET yyyy=%date:~,4%
SET MM=%date:~5,2%
SET dd=%date:~8,2%
SET yyyyMMdd=%yyyy%%MM%%dd%
REM 把年月日串中的空格替换为0
SET yyyyMMdd=%yyyyMMdd: =%

REM 根据当前时间获取，时分秒串
SET HH=%time:~0,2%
SET mm=%time:~3,2%
SET ss=%time:~6,2%
SET HHmmss=%HH%%mm%%ss%

SET timestamp=%yyyyMMdd%%HHmmss%

SET GOOS=linux
SET GOARCH=amd64

echo build start
go build -ldflags "-X main.Version=v0.1.0 -X main.Build=%timestamp%" -o ./bin/
echo build finish.