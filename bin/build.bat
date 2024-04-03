@echo off
cd ../

go env -w GO111MODULE=on
go env -w GOPROXY=https://goproxy.cn,direct

for /f "tokens=2 delims==" %%a in ('wmic OS Get localdatetime /value') do set "dt=%%a"
set "YY=%dt:~2,2%" & set "YYYY=%dt:~0,4%" & set "MM=%dt:~4,2%" & set "DD=%dt:~6,2%"
set "HH=%dt:~8,2%" & set "Min=%dt:~10,2%" & set "Sec=%dt:~12,2%"
set "timestamp=%YYYY%%MM%%DD%%HH%%Min%%Sec%"

SET GOOS=linux
SET GOARCH=amd64

echo build start
go build -ldflags "-X main.Version=v0.1.1 -X main.Build=%timestamp%" -o ./bin/
echo build finish.