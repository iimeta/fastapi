// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish manually maintain this interface file.
// ================================================================================

package service

import (
	"github.com/gogf/gf/v2/net/ghttp"
)

type (
	IRealtime interface {
		// Realtime
		Realtime(r *ghttp.Request) error
	}
)

var (
	localRealtime IRealtime
)

func Realtime() IRealtime {
	if localRealtime == nil {
		panic("implement not found for interface IRealtime, forgot register?")
	}
	return localRealtime
}

func RegisterRealtime(i IRealtime) {
	localRealtime = i
}
