package util

import (
	"github.com/gogf/gf/v2/os/gtime"
	"time"
)

const (
	DateDayFormat = "20060102"
)

func DateNumber() string {
	return time.Now().Format(DateDayFormat)
}

func Location() *time.Location {
	lo, _ := time.LoadLocation("Asia/Shanghai")
	return lo
}

func FormatDatetime(timestamp int64) string {
	return gtime.NewFromTimeStamp(timestamp).String()
}

func IsDateFormat(date string) bool {
	_, err := time.Parse(time.DateTime, date)
	return err != nil
}
